package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/nestoca/joy/api/v1alpha1"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func ProjectReconciler() ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, e ctrl.Event) (ctrl.Result, error) {
			releaseCache := ctrl.Cache[v1alpha1.Release](ctx, v1alpha1.ReleaseGK, metav1.NamespaceAll)

			releases, err := releaseCache.List(labels.Everything())
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list cached releases: %w", err)
			}

			releases = slices.DeleteFunc(releases, func(release *v1alpha1.Release) bool {
				return release.Spec.Project != e.Name
			})

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
