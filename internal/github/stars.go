package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log2 "github.com/apex/log"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

var (
	errNoMorePages  = errors.New("no more pages to get")
	ErrTooManyStars = errors.New("repo has too many stargazers, github won't allow to list all stars")
)

type Stargazer struct {
	StarredAt time.Time `json:"starred_at"`
}

func (gh *GitHub) Stargazers(ctx context.Context, repo Repository) (stars []Stargazer, err error) {
	if gh.totalPages(repo) > 400 {
		return stars, ErrTooManyStars
	}

	var (
		wg   errgroup.Group
		lock sync.Mutex
	)

	wg.SetLimit(4)
	for page := 1; page <= gh.lastPage(repo); page++ {
		page := page
		wg.Go(func() error {
			result, err := gh.getStarGazersPage(ctx, repo, page)
			if errors.Is(err, errNoMorePages) {
				return nil
			}
			if err != nil {
				return err
			}
			lock.Lock()
			defer lock.Unlock()
			stars = append(stars, result...)
			return nil
		})
	}

	err = wg.Wait()

	sort.Slice(stars, func(i, j int) bool {
		return stars[i].StarredAt.Before(stars[j].StarredAt)
	})
	return
}

func (gh *GitHub) getStarGazersPage(ctx context.Context, repo Repository, page int) ([]Stargazer, error) {
	log := log2.WithField("repo", repo.FullName).WithField("page", page)
	defer log.Trace("get page").Stop(nil)
	var stars []Stargazer
	key := fmt.Sprintf("%s_%d", repo.FullName, page)
	etagKey := fmt.Sprintf("%s_%d", repo.FullName, page) + "_etag"

	var etag string
	if err := gh.cache.Get(etagKey, &etag); err != nil {
		log.WithError(err).Warnf("failed to get %s from cache", etagKey)
	}

	resp, err := gh.makeStarPageRequest(ctx, repo, page, etag)
	if err != nil {
		return stars, nil
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return stars, nil
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		effectiveEtags.Inc()
		log.Info("not modified")
		err := gh.cache.Get(key, &stars)
		if err != nil {
			log.WithError(err).Warnf("failed to get %s from cache", key)
			if err := gh.cache.Delete(etagKey); err != nil {
				log.WithError(err).Warnf("failed to delete %s from cache", etagKey)
			}
			return gh.getStarGazersPage(ctx, repo, page)
		}
		return stars, err

	case http.StatusForbidden:
		rateLimits.Inc()
		log.Warn("rate limit hit")
		return stars, ErrRateLimit

	case http.StatusOK:
		if err := json.Unmarshal(bts, &stars); err != nil {
			return stars, err
		}
		if len(stars) == 0 {
			return stars, errNoMorePages
		}
		if err := gh.cache.Put(key, stars); err != nil {
			log.WithError(err).Warnf("failed to cache %s", key)
		}

		etag = resp.Header.Get("etag")
		if etag != "" {
			if err := gh.cache.Put(etagKey, etag); err != nil {
				log.WithError(err).Warnf("failed to cache %s", etagKey)
			}
		}
		return stars, nil
	default:
		return stars, fmt.Errorf("%w: %v", errGitHubAPI, string(bts))
	}
}

func (gh *GitHub) totalPages(repo Repository) int {
	return repo.StargazersCount / gh.pageSize
}

func (gh *GitHub) lastPage(repo Repository) int {
	return gh.totalPages(repo) + 1
}

func (gh *GitHub) makeStarPageRequest(ctx context.Context, repo Repository, page int, etag string) (*http.Response, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/stargazers?page=%d&per_page=%d",
		repo.FullName,
		page,
		gh.pageSize)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/vnd.github.v3.star+json")
	if etag != "" {
		req.Header.Add("If-None-Match", etag)
	}
	return gh.authorizedDo(req, 0)
}
