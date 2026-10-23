package main

import (
	"context"
	"encoding/json/v2"
	"log/slog"

	"github.com/yokecd/yoke/pkg/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	kcache "k8s.io/client-go/tools/cache"

	"github.com/nestoca/joy-operator/cmd/operator/helm"
)

type WatchRegistriesParams struct {
	Client    *k8s.Client
	Logger    *slog.Logger
	Namespace string
}

func WatchRegistries(ctx context.Context, params WatchRegistriesParams) func() {
	factory := informers.NewSharedInformerFactoryWithOptions(
		params.Client.Clientset,
		0,
		informers.WithNamespace(params.Namespace),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = metav1.FormatLabelSelector(&metav1.LabelSelector{
				MatchLabels: map[string]string{"joy.nesto.ca/dockerconfigjson": "helm"},
			})
			opts.FieldSelector = fields.OneTermEqualSelector("type", string(corev1.SecretTypeDockerConfigJson)).String()
		}),
	)

	login := func(obj any) {
		secret, _ := obj.(*corev1.Secret)
		if secret == nil {
			return
		}

		var dockerCfg helm.DockerConfig
		if err := json.Unmarshal(secret.Data[".dockerconfigjson"], &dockerCfg); err != nil {
			params.Logger.LogAttrs(
				ctx,
				slog.LevelError,
				"failed to parse dockerconfig",
				slog.String("error", err.Error()),
				slog.GroupAttrs("secret", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace)),
			)
			return
		}

		for registry, cfg := range dockerCfg.Auths {
			if err := helm.Login(ctx, helm.LoginParams{
				User:     cfg.Username,
				Password: []byte(cfg.Password),
				Registry: registry,
			}); err != nil {
				params.Logger.LogAttrs(
					ctx,
					slog.LevelError,
					"failed to authenticate with registry",
					slog.String("registry", registry),
					slog.String("error", err.Error()),
					slog.GroupAttrs("secret", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace)),
				)
				continue
			}
			params.Logger.Info("successfully authenticated with registry", "registry", registry)
		}
	}

	logout := func(obj any) {
		secret, _ := obj.(*corev1.Secret)
		if secret == nil {
			return
		}

		var dockerCfg helm.DockerConfig
		if err := json.Unmarshal(secret.Data[".dockerconfigjson"], &dockerCfg); err != nil {
			params.Logger.LogAttrs(
				ctx,
				slog.LevelError,
				"failed to parse dockerconfig",
				slog.String("error", err.Error()),
				slog.GroupAttrs("secret", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace)),
			)
			return
		}

		for registry := range dockerCfg.Auths {
			if err := helm.Logout(ctx, registry); err != nil {
				params.Logger.LogAttrs(
					ctx,
					slog.LevelError,
					"failed to logout from registry",
					slog.String("registry", registry),
					slog.String("error", err.Error()),
					slog.GroupAttrs("secret", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace)),
				)
				continue
			}
			params.Logger.Info("successfully logged out from registry", "registry", registry)
		}
	}

	factory.Core().V1().Secrets().Informer().AddEventHandler(kcache.ResourceEventHandlerFuncs{
		AddFunc:    login,
		UpdateFunc: func(_, obj any) { login(obj) },
		DeleteFunc: logout,
	})

	params.Logger.Info("watching for registry secrets")

	ctx, cancel := context.WithCancel(ctx)

	factory.StartWithContext(ctx)
	factory.WaitForCacheSyncWithContext(ctx)

	return func() {
		cancel()
		factory.Shutdown()
	}
}
