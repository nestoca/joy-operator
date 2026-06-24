package main

import (
	"context"
	"fmt"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
	"github.com/nestoca/joy/api/v1alpha1"
)

type EnvironmentReconcilerParams struct {
	CatalogName string
	Pull        bool
}

func EnvironmentReconciler(params EnvironmentReconcilerParams) ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			var (
				client       = ctrl.Client(ctx)
				envCache     = ctrl.CacheFromEvent[v1alpha1.Environment](ctx, event)
				catalogCache = ctrl.Cache[v1alpha1.Catalog](ctx, v1alpha1.CatalogGK, "")
				nsIntf       = k8s.TypedInterface[corev1.Namespace](client.Dynamic, schema.GroupVersionResource{
					Version:  "v1",
					Resource: "namespaces",
				})
				appIntf = k8s.TypedInterface[argocd.Application](client.Dynamic, argocd.ApplicationGVR).Namespace("argocd")
			)

			env, err := envCache.Get(event.Name)
			if err != nil {
				if kerrors.IsNotFound(err) {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, err
			}

			catalog, err := catalogCache.Get(params.CatalogName)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get catalog: %w", err)
			}

			ns := &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   env.Name,
					Labels: map[string]string{"nesto.ca/env": env.Name},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "joy.nesto.ca/v1alpha1",
							Kind:       v1alpha1.EnvironmentKind,
							Name:       env.Name,
							UID:        env.UID,
						},
					},
				},
			}

			if _, err = nsIntf.Apply(ctx, ns, metav1.ApplyOptions{FieldManager: joyOperator}); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to apply namespace: %w", err)
			}

			if params.Pull {
				if _, err := appIntf.Apply(
					ctx,
					&argocd.Application{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "argoproj.io/v1alpha1",
							Kind:       "Application",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      event.Name,
							Namespace: "argocd",
						},
						Spec: argocd.ApplicationSpec{
							Project: "default",
							Source: argocd.ApplicationSource{
								RepoURL:        catalog.Spec.RepoURL,
								TargetRevision: "master",
								Directory: argocd.SourceDirectory{
									Include: fmt.Sprintf("environments/%s/releases", env.Name),
									Recurse: true,
								},
							},
							Destination: argocd.ApplicationDestination{
								Server:    "http://kubernetes.svc.local",
								Namespace: ns.Name,
							},
						},
					},
					metav1.ApplyOptions{FieldManager: joyOperator},
				); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to apply application: %w", err)
				}
			} else {
				if err := appIntf.Delete(ctx, event.Name, metav1.DeleteOptions{}); err != nil && !kerrors.IsNotFound(err) {
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
