package controller

import (
	"fmt"
	"github.com/apex/log"
	"github.com/caarlos0/httperr"
	"io"
	"net/http"
	"strarcharts/internal/cache"
	"strarcharts/internal/chart"
	"strarcharts/internal/chart/svg"
	"strarcharts/internal/github"
	"strings"
	"time"
)

var stylesMap = map[string]string{
	"light":    chart.LightStyles,
	"dark":     chart.DarkStyles,
	"adaptive": chart.AdaptiveStyles,
}

func GetRepoChart(gh *github.GitHub, cache *cache.Redis) http.Handler {
	return httperr.NewF(func(w http.ResponseWriter, r *http.Request) error {
		params, err := extractSvgChartParams(r)
		if err != nil {
			log.WithError(err).Error("failed to extract params")
			return err
		}

		cacheKey := chartKey(params)
		name := fmt.Sprintf("%s/%s", params.Owner, params.Repo)
		log := log.WithField("repo", name).WithField("variant", params.Variant)

		cacheChart := ""
		if err = cache.Get(cacheKey, &cacheChart); err == nil {
			writeSvgHeaders(w)
			log.Debug("using cached chart")
			_, err := fmt.Fprint(w, cacheChart)
			return err
		}

		defer log.Trace("collect_stars").Stop(nil)
		repo, err := gh.RepoDetails(r.Context(), name)

		if err != nil {
			return httperr.Wrap(err, http.StatusBadRequest)
		}
		stargazers, err := gh.Stargazers(r.Context(), repo)
		if err != nil {
			log.WithError(err).Error("failed to get stars")
			writeSvgHeaders(w)
			_, err = w.Write([]byte(errSvg(err)))
			return err
		}

		series := chart.Series{
			StrokeWidth: 2,
			Color:       params.Line,
		}
		for i, star := range stargazers {
			series.XValues = append(series.XValues, star.StarredAt)
			series.YValues = append(series.YValues, float64(i+1))
		}
		if len(series.XValues) < 2 {
			log.Info("not enough results, adding some fake ones")
			series.XValues = append(series.XValues, time.Now())
			series.YValues = append(series.YValues, 1)
		}

		graph := &chart.Chart{
			Width:      CHART_WIDTH,
			Height:     CHART_HEIGHT,
			Styles:     stylesMap[params.Variant],
			Background: params.Background,
			XAxis: chart.XAxis{
				Name:        "Time",
				Color:       params.Axis,
				StrokeWidth: 2,
			},
			YAxis: chart.YAxis{
				Name:        "Stargazers",
				Color:       params.Axis,
				StrokeWidth: 2,
			},
			Series: series,
		}
		defer log.Trace("chart").Stop(&err)

		writeSvgHeaders(w)

		cacheBuffer := &strings.Builder{}
		graph.Render(io.MultiWriter(w, cacheBuffer))
		err = cache.Put(cacheKey, cacheBuffer.String())
		if err != nil {
			log.WithError(err).Error("failed to cache chart")
		}

		return nil
	})
}

func errSvg(err error) string {
	return svg.SVG().
		Attr("width", svg.Px(CHART_WIDTH)).
		Attr("height", svg.Px(CHART_HEIGHT)).
		ContentFunc(func(writer io.Writer) {
			svg.Text().
				Attr("fill", "red").
				Attr("x", svg.Px(CHART_WIDTH/2)).
				Attr("y", svg.Px(CHART_HEIGHT/2)).
				Content(err.Error()).
				Render(writer)
		}).
		String()
}
