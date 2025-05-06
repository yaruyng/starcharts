package github

import (
	"errors"
	"strarcharts/roundrobin"
)

var ErrRateLimit = errors.New("rate limited, please try again later")

var errGitHubAPI = errors.New("failed to talk with github api")

type GitHub struct {
	tokens roundrobin.RoundRobiner
}
