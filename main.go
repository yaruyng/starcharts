package main

import (
	"embed"
	"github.com/apex/httplog"
	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	config2 "strarcharts/config"
	"strarcharts/controller"
	"strarcharts/internal/cache"
	github2 "strarcharts/internal/github"
	"time"
)

//go:embed static/*
var static embed.FS

//This line of code `//go:embed static/*` is a special directive in Go language, used to embed static files into the Go program.

var version = "devel"

func main() {
	log.SetHandler(text.New(os.Stderr))
	config := config2.Get()
	ctx := log.WithField("listen", config.Listen)
	options, err := redis.ParseURL(config.RedisUrl)
	if err != nil {
		log.WithError(err).Fatal("invalid redis_url")
	}
	redis := redis.NewClient(options)
	cache := cache.New(redis)
	defer cache.Close()
	github := github2.New(config, cache)

	r := mux.NewRouter()
	r.Path("/").
		Methods(http.MethodGet).
		Handler(controller.Index(static, version))
	r.Path("/").
		Methods(http.MethodPost).
		HandlerFunc(controller.HandleForm())
	r.PathPrefix("/static/").
		Methods(http.MethodGet).
		Handler(http.FileServer(http.FS(static)))
	r.Path("/{owner}/{repo}.svg").
		Methods(http.MethodGet).
		Handler(controller.GetRepoChart(github, cache))
	r.Path("/{owner}/{repo}").
		Methods(http.MethodGet).
		Handler(controller.GetRepo(static, github, cache, version))
	// generic metrics
	requestCounter := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "starcharts",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "total requests",
	}, []string{"code", "method"})
	responseObserver := promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "starcharts",
		Subsystem: "http",
		Name:      "responses",
		Help:      "response times and counts",
	}, []string{"code", "method"})

	r.Methods(http.MethodGet).Path("/metrics").Handler(promhttp.Handler())

	srv := &http.Server{
		Handler: httplog.New(
			promhttp.InstrumentHandlerDuration(
				responseObserver,
				promhttp.InstrumentHandlerCounter(
					requestCounter,
					r,
				),
			),
		),
		Addr:         config.Listen,
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}
	ctx.Info("starting up...")
	ctx.WithError(srv.ListenAndServe()).Error("failed to start up server")
}
