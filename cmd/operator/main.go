package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/davidmdm/x/xcontext"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/nestoca/joy/api/v1alpha1"
	joy "github.com/nestoca/joy/pkg"
	"github.com/nestoca/joy/pkg/helm"
)

const joyOperator = "joy-operator"

func main() {
	if run() != nil {
		os.Exit(1)
	}
}

func run() (err error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	defer func() {
		if err != nil {
			logger.Error("exiting with error", "error", err.Error())
		}
	}()

	ctx, cancel := xcontext.WithSignalCancelation(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if cfg.HelmLogin.Registry != "" {
		if err := AuthenticateHelm(ctx, cfg.HelmLogin.Registry, cfg.HelmLogin.Credentials); err != nil {
			return fmt.Errorf("failed to authenticate helm: %w", err)
		}
	}

	restCfg, err := func() (*rest.Config, error) {
		if cfg, err := rest.InClusterConfig(); err == nil {
			return cfg, err
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube/config"))
	}()
	if err != nil {
		return fmt.Errorf("failed to build kubernetes rest config: %w", err)
	}

	client, err := k8s.NewClient(restCfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	controller := ctrl.NewController(ctrl.Params{
		Client:      client,
		Logger:      logger,
		Concurrency: cfg.Concurrency,
	})

	if err := controller.Register(
		ctrl.Entry{
			GroupKind: v1alpha1.EnvironmentGK,
			Funcs: EnvironmentReconciler(EnvironmentReconcilerParams{
				CatalogName: cfg.CatalogName,
				Pull:        cfg.Pull,
			}),
		},
		ctrl.Entry{
			GroupKind: v1alpha1.ReleaseGK,
			Funcs: ReleaseReconciler(ReleaseReconcilerParams{
				ChartSource: ChartSource{
					Root:   cfg.ChartCacheDir,
					Puller: helm.CLI{IO: joy.IO{Out: os.Stdout, Err: os.Stderr}},
				},
				EnvDestinations: cfg.EnvDestinations,
				CatalogName:     cfg.CatalogName,
			}),
		},
		ctrl.Entry{
			GroupKind: v1alpha1.ProjectGK,
			Funcs: ctrl.Funcs{
				Handler: func(ctx context.Context, e ctrl.Event) (ctrl.Result, error) {
					// We do not want to do anything with projects other than have them stored in etcd.
					// We attach a noop reconciler to the resource simply so that they can be available to other reconcilers via the informer cache.
					return ctrl.Result{}, nil
				},
			},
		},
		ctrl.Entry{
			GroupKind: v1alpha1.CatalogGK,
			Funcs: CatalogReconciler(CatalogReconcilerParams{
				CatalogName:      cfg.CatalogName,
				EnvSourcePattern: cfg.EnvSourcePattern,
				Pull:             cfg.Pull,
			}),
		},
	); err != nil {
		return fmt.Errorf("failed to register reconcilers: %w", err)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	e := make(chan error, 2)

	wg.Go(func() {
		logger.Info("starting controller")
		if err := controller.Run(ctx); err != nil {
			e <- err
		}
	})

	wg.Go(func() {
		svr := http.Server{
			Addr: ":3000",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ready" && r.Method == "GET" {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusNotImplemented)
			}),
		}

		serverErr := make(chan error)
		go func() {
			logger.Info("starting server")
			if err := svr.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		}()

		select {
		case err := <-serverErr:
			e <- fmt.Errorf("failed to start server: %w", err)
			return
		case <-ctx.Done():
			logger.Info("server context canceled", "cause", context.Cause(ctx).Error())
		}

		// TODO: make graceful shutdown period configurable
		ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, fmt.Errorf("exceeded graceful period timeout"))
		defer cancel()

		if err := svr.Shutdown(ctx); err != nil {
			e <- fmt.Errorf("failed to shutdown: %w", err)
		}
	})

	go func() {
		wg.Wait()
		close(e)
	}()

	return <-e
}

func AuthenticateHelm(ctx context.Context, registry string, credentials []byte) error {
	login := exec.CommandContext(ctx, "helm", "registry", "login", "-u", "_json_key", "--password-stdin", registry)

	var buffer bytes.Buffer
	login.Stdout = &buffer
	login.Stderr = &buffer
	login.Stdin = bytes.NewReader(credentials)

	if err := login.Run(); err != nil {
		return fmt.Errorf("%w: %q", err, &buffer)
	}

	return nil
}
