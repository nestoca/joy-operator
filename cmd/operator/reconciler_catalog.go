package main

import (
	"context"
	"fmt"

	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/nestoca/joy/api/v1alpha1"
)

func CatalogReconciler() ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
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
