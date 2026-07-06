package main

import (
	"context"
	"fmt"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/nestoca/joy-operator/cmd/operator/argocd"
	"github.com/nestoca/joy/api/v1alpha1"
)

type CatalogReconcilerParams struct {
	CatalogName string
	Pull        bool
}

func CatalogReconciler(params CatalogReconcilerParams) ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			if event.Name != params.CatalogName {
				return ctrl.Result{}, ctrl.Terminalf("unsupported catalog: wanted %q got %q", params.CatalogName, event.Name)
			}

			catalogCache := ctrl.CacheFromEvent[v1alpha1.Catalog](ctx, event)

			catalog, err := catalogCache.Get(event.Name)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get catalog: %w", err)
			}

			appIntf := k8s.TypedInterface[argocd.Application](ctrl.Client(ctx), argocd.ApplicationGVR).Namespace("argocd")

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
								TargetRevision: catalog.Spec.Revision,
								Directory:      argocd.SourceDirectory{Include: "environments/*/env.yaml"},
							},
							Destination: argocd.ApplicationDestination{
								Server: "http://kubernetes.svc.local",
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

			envCache := ctrl.Cache[v1alpha1.Environment](ctx, v1alpha1.EnvironmentGK, "")

			envs, err := envCache.List(labels.Everything())
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list cached environments: %w", err)
			}

			for _, env := range envs {
				ctrl.Inst(ctx).SendEvent(ctrl.Event{
					Name:      env.Name,
					Namespace: env.Namespace,
					GroupKind: v1alpha1.EnvironmentGK,
				})
			}

			return ctrl.Result{}, nil
		},
	}
}
