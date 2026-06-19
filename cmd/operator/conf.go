package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/davidmdm/conf"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
)

type Config struct {
	ChartCacheDir   string
	EnvDestinations map[string]argocd.ApplicationDestination
	HelmLogin       HelmLogin
	Concurrency     int
}

type HelmLogin struct {
	Registry        string
	CredentialsPath string
	Credentials     []byte
}

var defaultConcurrency = max(4, runtime.GOMAXPROCS(-1))

func GetConfig() (Config, error) {
	var cfg Config

	conf.Var(conf.Environ, &cfg.EnvDestinations, "ENV_DESTINATIONS", conf.JSON[map[string]argocd.ApplicationDestination])
	conf.Var(conf.Environ, &cfg.HelmLogin.Registry, "HELM_REGISTRY")
	conf.Var(conf.Environ, &cfg.HelmLogin.CredentialsPath, "HELM_REGISTRY_CREDENTIALS_PATH")
	conf.Var(conf.Environ, &cfg.ChartCacheDir, "CHART_CACHE_DIR", conf.RequiredNonEmpty[string]())
	conf.Var(conf.Environ, &cfg.Concurrency, "CONCURRENCY", conf.Default(defaultConcurrency))

	if err := conf.Environ.Parse(); err != nil {
		return cfg, fmt.Errorf("failed to parse environment: %w", err)
	}

	cfg.Concurrency = max(cfg.Concurrency, 1)

	if cfg.HelmLogin.Registry != "" {
		var err error
		cfg.HelmLogin.Credentials, err = os.ReadFile(cfg.HelmLogin.CredentialsPath)
		if err != nil {
			return cfg, fmt.Errorf("failed to load helm registry credentials: %w", err)
		}
	}

	return cfg, nil
}
