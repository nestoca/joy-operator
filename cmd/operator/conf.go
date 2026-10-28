package main

import (
	"fmt"
	"runtime"

	"github.com/davidmdm/conf"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
)

type Config struct {
	CatalogName                     string
	ChartCacheDir                   string
	EnvDestinations                 map[string]argocd.ApplicationDestination
	EnvSourcePattern                string
	Concurrency                     int
	Pull                            bool
	ServiceName                     string
	Streams                         []string
	ServiceNamespace                string
	ApplicationRevisionHistoryLimit int
}

var defaultConcurrency = max(4, runtime.GOMAXPROCS(-1))

func GetConfig() (Config, error) {
	var cfg Config

	conf.Environ.Var(&cfg.EnvDestinations, "ENV_DESTINATIONS", conf.JSON[map[string]argocd.ApplicationDestination])
	conf.Environ.Var(&cfg.EnvSourcePattern, "ENV_SOURCE_PATTERN", conf.Default("environments/*/env.yaml"))
	conf.Environ.Var(&cfg.ChartCacheDir, "CHART_CACHE_DIR", conf.RequiredNonEmpty[string]())
	conf.Environ.Var(&cfg.Concurrency, "CONCURRENCY", conf.Default(defaultConcurrency))
	conf.Environ.Var(&cfg.Pull, "PULL_MODE")
	conf.Environ.Var(&cfg.CatalogName, "DEFAULT_CATALOG", conf.Default("catalog"))
	conf.Environ.Var(&cfg.Streams, "STREAMS")
	conf.Environ.Var(&cfg.ServiceName, "SERVICE_NAME", conf.Default("joy-operator-dev"))
	conf.Environ.Var(&cfg.ServiceNamespace, "SERVICE_NAMESPACE", conf.RequiredNonEmpty[string]())
	conf.Environ.Var(&cfg.ApplicationRevisionHistoryLimit, "APPLICATION_REVISION_HISTORY_LIMIT", conf.Default(3))

	if err := conf.Environ.Parse(); err != nil {
		return cfg, fmt.Errorf("failed to parse environment: %w", err)
	}

	return cfg, nil
}
