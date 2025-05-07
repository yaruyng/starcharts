package github

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"strarcharts/config"
	"strarcharts/internal/cache"
	"strarcharts/roundrobin"
)

var ErrRateLimit = errors.New("rate limited, please try again later")

var errGitHubAPI = errors.New("failed to talk with github api")

type GitHub struct {
	tokens          roundrobin.RoundRobiner
	pageSize        int
	cache           *cache.Redis
	maxRateUsagePct int
}

var rateLimits = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: "starcharts",
	Subsystem: "github",
	Name:      "rate_limit_hits_total",
})

var effectiveEtags = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: "starcharts",
	Subsystem: "github",
	Name:      "effective_etag_uses_total",
})

var tokensCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "starcharts",
	Subsystem: "github",
	Name:      "available_tokens",
})

var invalidatedTokens = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "starcharts",
	Subsystem: "github",
	Name:      "invalidated_tokens_total",
})

var rateLimiters = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "starcharts",
	Subsystem: "github",
	Name:      "rate_limit_remaining",
}, []string{"token"})

func init() {
	prometheus.MustRegister(rateLimits, effectiveEtags, invalidatedTokens, tokensCount, rateLimiters)
}

func New(config config.Config, cache *cache.Redis) *GitHub {
	tokensCount.Set(float64(len(config.GithubTokens)))
	return &GitHub{
		tokens:   roundrobin.New(config.GithubTokens),
		pageSize: config.GithubPageSize,
		cache:    cache,
	}
}

const maxTries = 3

//func (gh *GitHub) authorizedDo(req *http.Request, try int) (*http.Response, error) {
//
//}
