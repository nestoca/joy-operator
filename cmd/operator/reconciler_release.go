package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nestoca/joy/api/v1alpha1"
	joy "github.com/nestoca/joy/pkg"
	"github.com/nestoca/joy/pkg/helm"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
)

type ReleaseReconcilerParams struct {
	ChartCache      helm.ChartCache
	EnvDestinations map[string]argocd.ApplicationDestination
}

func ReleaseReconciler(params ReleaseReconcilerParams) ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			var (
				releaseCache = ctrl.CacheFromEvent[v1alpha1.Release](ctx, event)
				envCache     = ctrl.Cache[v1alpha1.Environment](ctx, schema.GroupKind{Group: "joy.nesto.ca", Kind: v1alpha1.KindEnvironment}, "")
				projectCache = ctrl.Cache[v1alpha1.Project](ctx, schema.GroupKind{Group: "joy.nesto.ca", Kind: v1alpha1.KindProject}, "")
				client       = ctrl.Client(ctx)
				appIntf      = k8s.TypedInterface[argocd.Application](client.Dynamic, argocd.ApplicationGVR).Namespace("argocd")
			)

			release, err := releaseCache.Get(event.Name)
			if err != nil {
				if kerrors.IsNotFound(err) {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, fmt.Errorf("failed to get release: %w", err)
			}

			release.Environment, err = envCache.Get(release.Namespace)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get environment: %w", err)
			}

			release.Project, err = projectCache.Get(release.Spec.Project)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get project: %w", err)
			}

			destination, ok := params.EnvDestinations[release.Environment.Name]
			if !ok {
				return ctrl.Result{}, ctrl.Terminal(fmt.Errorf("no app destination found for environment %s", release.Environment.Name))
			}

			chartFS, err := params.ChartCache.GetReleaseChartFS(ctx, release)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get release chart filesystem: %w", err)
			}

			values, err := joy.ComputeReleaseValues(release, chartFS)
			if err != nil {
				return ctrl.Result{}, ctrl.Terminal(fmt.Errorf("failed to compute release values: %w", err))
			}

			valuesBytes, err := yaml.Marshal(values)
			if err != nil {
				return ctrl.Result{}, ctrl.Terminal(fmt.Errorf("failed to marshal release values: %w", err))
			}

			app := renderReleaseApplication(RenderApplicationParams{
				Release:     release,
				Destination: destination,
				Values:      valuesBytes,
				Chart:       chartFS.Chart,
			})

			if _, err = appIntf.Apply(ctx, &app, metav1.ApplyOptions{FieldManager: joyOperator}); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to apply application: %w", err)
			}

			return ctrl.Result{}, nil
		},
	}
}

type RenderApplicationParams struct {
	Release     *v1alpha1.Release
	Destination argocd.ApplicationDestination
	Values      []byte
	Chart       helm.Chart
}

func renderReleaseApplication(params RenderApplicationParams) argocd.Application {
	//  In the future we may want to make the joy-operator less nesto-specific.
	//  The current implementation is a direct mapping of an internal nesto argocd AppSet that the operator would deprecate.
	//  Ideally labels and annotations should be controlled at the release level or configurable by the operator.
	//  Allowing nesto to create the labels/annotations it needs, but also make it viable as a standalone project.
	//
	//  Also we may wish to make the labels and annotations joy specific.
	//  IE:
	//
	// 		Labels: copyMaps(
	// 		params.Release.Labels,
	// 		map[string]string{
	// 			"joy.nesto.ca/release":    params.Release.Name,
	// 			"joy.nesto.ca/env":        params.Release.Environment.Name,
	// 			"joy.nesto.ca/project":    params.Release.Project.Name,
	// 			"joy.nesto.ca/repository": params.Release.Project.Spec.Repository,
	// 			"joy.nesto.ca/owner": func() string {
	// 				if len(params.Release.Project.Spec.Owners) == 0 {
	// 					return ""
	// 				}
	// 				return params.Release.Environment.Spec.Owners[0]
	// 			}(),
	// 		},
	// 	),
	// 	Annotations: copyMaps(
	// 		knownAnnotationMappings(params.Release.Annotations), // TODO: maybe axe this concept?
	// 		params.DefaultAnnotations,
	// 		params.Release.Annotations,
	// 		map[string]string{"joy.nesto.ca/version": params.Release.Spec.Version},
	// 	),
	// },

	return argocd.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", params.Release.Environment.Name, params.Release.Name),
			Namespace: "argocd",
			Labels: map[string]string{
				"nesto.ca/release":    "true",
				"nesto.ca/env":        params.Release.Environment.Name,
				"nesto.ca/project":    params.Release.Name,
				"nesto.ca/version":    params.Release.Spec.Version,
				"nesto.ca/repository": params.Release.Project.Spec.Repository,
				"nesto.ca/stream": func() string {
					streams := []string{"origination", "servicing", "platform", "marketing", "cross-system", "security", "data-engineering"}
					for _, owner := range params.Release.Project.Spec.Owners {
						if slices.Contains(streams, owner) {
							return owner
						}
					}
					return "lost"
				}(),
			},
			Annotations: map[string]string{
				"nesto.ca/release-version":                                        params.Release.Spec.Version,
				"notifications.argoproj.io/subscribe.on-production-release.slack": "notif-releases",
				"notifications.argoproj.io/subscribe.on-release.slack":            params.Release.Annotations["nesto.ca/notifications-channel"],
			},
		},
		Spec: argocd.ApplicationSpec{
			SyncPolicy: argocd.SyncPolicy{
				SyncOptions: []string{"CreateNamespace=true"},
				Automated: func() argocd.SyncPolicyAutomated {
					if params.Release.Annotations["argocd.nesto.ca/sync.enabled"] != "true" {
						return argocd.SyncPolicyAutomated{}
					}
					return argocd.SyncPolicyAutomated{
						Prune:    new(params.Release.Annotations["argocd.nesto.ca/sync.prune"] == "true"),
						SelfHeal: new(params.Release.Annotations["argocd.nesto.ca/sync.heal"] == "true"),
					}
				}(),
			},
			Project: cmp.Or(params.Release.Environment.Annotations["joy.nesto.ca/argocd.project"], "default"),
			Source: argocd.ApplicationSource{
				Chart:          params.Chart.Name,
				RepoURL:        params.Chart.RepoURL,
				TargetRevision: params.Chart.Version,
				Helm: argocd.SourceHelm{
					ReleaseName: params.Release.Name,
					Values:      string(params.Values),
				},
			},
			Destination: params.Destination,
		},
	}
}
