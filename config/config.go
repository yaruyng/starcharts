package config

import (
	"github.com/apex/log"
	"github.com/caarlos0/env/v6"
)

// Config configuration.
type Config struct {
	RedisUrl              string   `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	GithubTokens          []string `env:"GITHUB_TOKENS"`
	GithubPageSize        int      `env:"GITHUB_PAGE_SIZE" envDefault:"100"`
	GitHubMaxRateUsagePct int      `env:"GITHUB_MAX_RATE_LIMIT_USAGE" envDefault:"80"`
	Listen                string   `env:"LISTEN" envDefault:"127.0.0.1:3000"`
}

// Get the current Config.
func Get() (cfg Config) {
	if err := env.Parse(&cfg); err != nil {
		log.WithError(err).Fatal("failed to load config")
	}
	return
}
