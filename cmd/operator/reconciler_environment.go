package main

import (
	"context"
	"fmt"

	"github.com/davidmdm/x/xcontainer"

	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"

	"github.com/nestoca/joy/api/v1alpha1"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
)

type EnvironmentReconcilerParams struct {
	CatalogName string
	Pull        bool
	ManagedEnvs xcontainer.Set[string]
}

func EnvironmentReconciler(params EnvironmentReconcilerParams) ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			if !params.ManagedEnvs.Has(event.Name) {
				return ctrl.Result{}, nil
			}

			client := ctrl.Client(ctx)
			envCache := ctrl.CacheFromEvent[v1alpha1.Environment](ctx, event)

			env, err := envCache.Get(event.Name)
			if err != nil {
				if kerrors.IsNotFound(err) {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, err
			}

			catalogCache := ctrl.Cache[v1alpha1.Catalog](ctx, v1alpha1.CatalogGK, "")

			catalog, err := catalogCache.Get(params.CatalogName)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get catalog: %w", err)
			}

			ns := &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
				ObjectMeta: metav1.ObjectMeta{
					Name:   env.Name,
					Labels: map[string]string{"nesto.ca/env": env.Name},
				},
			}

			nsIntf := client.TypedInterface[corev1.Namespace](schema.GroupVersionResource{
				Version:  "v1",
				Resource: "namespaces",
			})

			if _, err = nsIntf.Apply(ctx, ns, metav1.ApplyOptions{FieldManager: joyOperator}); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to apply namespace: %w", err)
			}

			appIntf := client.TypedInterface[argocd.Application](argocd.ApplicationGVR).Namespace("argocd")

			if params.Pull {
				if _, err := appIntf.Apply(
					ctx,
					&argocd.Application{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "argoproj.io/v1alpha1",
							Kind:       "Application",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:       event.Name,
							Namespace:  "argocd",
							Finalizers: []string{"resources-finalizer.argocd.argoproj.io"},
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion:         env.ApiVersion,
									Kind:               env.Kind,
									Name:               env.Name,
									UID:                env.UID,
									Controller:         new(true),
									BlockOwnerDeletion: new(true),
								},
							},
						},
						Spec: argocd.ApplicationSpec{
							Project: "default",
							Source: argocd.ApplicationSource{
								RepoURL:        catalog.Spec.RepoURL,
								TargetRevision: catalog.Spec.Revision,
								Path:           "./",
								Directory: argocd.SourceDirectory{
									Include: fmt.Sprintf("environments/%s/releases/**/*.yaml", env.Name),
									Recurse: true,
								},
							},
							Destination: argocd.ApplicationDestination{
								Server:    "https://kubernetes.default.svc",
								Namespace: ns.Name,
							},
							SyncPolicy: argocd.SyncPolicy{
								Automated: &argocd.SyncPolicyAutomated{
									Prune: new(true),
								},
							},
						},
					},
					metav1.ApplyOptions{FieldManager: joyOperator, Force: true},
				); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to apply application: %w", err)
				}
			} else {
				if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
					app, err := appIntf.Get(ctx, event.Name, metav1.GetOptions{})
					if err != nil {
						return fmt.Errorf("failed to get application: %w", err)
					}
					app.Spec.SyncPolicy = argocd.SyncPolicy{
						Automated: &argocd.SyncPolicyAutomated{
							Prune: new(false),
						},
					}

					app, err = appIntf.Apply(ctx, app, metav1.ApplyOptions{FieldManager: joyOperator, Force: true})
					if err != nil {
						return fmt.Errorf("failed to update syncPolicy to not prune: %w", err)
					}
					// We want a guarantee that when we delete the application that it was against a version of the resource who has prune de-activated.
					// If we move from pull mode to non-pull mode, we don't want to drop all releases.
					return appIntf.Delete(ctx, app.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &app.ResourceVersion}})
				}); err != nil && !kerrors.IsNotFound(err) {
					return ctrl.Result{}, fmt.Errorf("failed to delete application: %w", err)
				}
			}

			releaseCache := ctrl.Cache[v1alpha1.Release](ctx, v1alpha1.ReleaseGK, ns.Name)

			releases, err := releaseCache.List(labels.Everything())
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list cached releases: %w", err)
			}

			for _, release := range releases {
				ctrl.Inst(ctx).SendEvent(ctrl.Event{
					Name:      release.Name,
					Namespace: release.Namespace,
					GroupKind: v1alpha1.ReleaseGK,
				})
			}

			return ctrl.Result{}, nil
		},
	}
}
