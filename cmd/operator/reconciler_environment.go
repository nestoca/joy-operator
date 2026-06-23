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

	"github.com/nestoca/joy/api/v1alpha1"
)

var releaseGK = schema.GroupKind{Group: "joy.nesto.ca", Kind: v1alpha1.ReleaseKind}

func EnvironmentReconciler() ctrl.Funcs {
	return ctrl.Funcs{
		Handler: func(ctx context.Context, event ctrl.Event) (ctrl.Result, error) {
			var (
				client   = ctrl.Client(ctx)
				envCache = ctrl.CacheFromEvent[v1alpha1.Environment](ctx, event)
				nsIntf   = k8s.TypedInterface[corev1.Namespace](client.Dynamic, schema.GroupVersionResource{
					Version:  "v1",
					Resource: "namespaces",
				})
			)

			env, err := envCache.Get(event.Name)
			if err != nil {
				if kerrors.IsNotFound(err) {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, err
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

			releaseCache := ctrl.Cache[v1alpha1.Release](ctx, releaseGK, ns.Name)

			releases, err := releaseCache.List(labels.Everything())
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list cached releases: %w", err)
			}

			for _, release := range releases {
				ctrl.Inst(ctx).SendEvent(ctrl.Event{
					Name:      release.Name,
					Namespace: release.Namespace,
					GroupKind: releaseGK,
				})
			}

			return ctrl.Result{}, nil
		},
	}
}
