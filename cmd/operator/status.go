package main

import (
	"context"

	"github.com/yokecd/yoke/pkg/k8s"
	"github.com/yokecd/yoke/pkg/k8s/ctrl"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nestoca/joy/api/v1alpha1"
)

func writeStatus[T any, PT v1alpha1.StatusObject[T]](
	ctx context.Context,
	gvr schema.GroupVersionResource,
	obj PT,
	reconcileErr error,
) {
	cond := metav1.Condition{
		Type:               v1alpha1.ConditionReady,
		ObservedGeneration: obj.GetGeneration(),
	}
	if reconcileErr != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = "ReconcileError"
		cond.Message = reconcileErr.Error()
	} else {
		cond.Status = metav1.ConditionTrue
		cond.Reason = "ReconcileSuccess"
		cond.Message = "Successfully reconciled"
	}

	status := obj.GetStatus()
	changed := apimeta.SetStatusCondition(&status.Conditions, cond)
	if status.ObservedGeneration != obj.GetGeneration() {
		status.ObservedGeneration = obj.GetGeneration()
		changed = true
	}

	// As the controller re-enqueues on every update, let's avoid an infinite reconcile loop.
	if !changed {
		return
	}

	if _, err := k8s.TypedInterface[T, PT](ctrl.Client(ctx), gvr).
		UpdateStatus(ctx, obj, metav1.UpdateOptions{FieldManager: joyOperator}); err != nil {
		if logger := ctrl.Logger(ctx); logger != nil {
			logger.Error("failed to update resource status", "gvr", gvr.String(), "name", obj.GetName(), "error", err.Error())
		}
	}
}
