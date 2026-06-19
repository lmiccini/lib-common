/*
Copyright 2020 Red Hat

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package statefulset provides utilities for managing Kubernetes StatefulSet resources
package statefulset

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/pod"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	appsv1 "k8s.io/api/apps/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewStatefulSet returns an initialized NewStatefulset.
func NewStatefulSet(
	statefulset *appsv1.StatefulSet,
	timeout time.Duration,
) *StatefulSet {
	return &StatefulSet{
		statefulset: statefulset,
		timeout:     timeout,
	}
}

// CreateOrPatch - creates or patches a statefulset, reconciles after Xs if object won't exist.
func (s *StatefulSet) CreateOrPatch(
	ctx context.Context,
	h *helper.Helper,
) (ctrl.Result, error) {
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.statefulset.Name,
			Namespace: s.statefulset.Namespace,
		},
	}

	log := ctrl.Log.WithName("statefulset-debug")
	op, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), statefulset, func() error {
		beforeU, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(statefulset.DeepCopy())

		statefulset.Labels = util.MergeStringMaps(statefulset.Labels, s.statefulset.Labels)
		statefulset.Annotations = util.MergeStringMaps(statefulset.Annotations, s.statefulset.Annotations)

		// Selector and VolumeClaimTemplates are immutable after creation.
		if !statefulset.CreationTimestamp.IsZero() {
			s.statefulset.Spec.Selector = statefulset.Spec.Selector
			s.statefulset.Spec.VolumeClaimTemplates = statefulset.Spec.VolumeClaimTemplates
		}

		pod.SetPullPolicyDefaults(&s.statefulset.Spec.Template.Spec)

		// Use strategic merge patch to apply the desired spec onto the
		// existing spec. This preserves all server-defaulted fields
		// automatically: fields the operator doesn't set (zero value,
		// omitted by JSON omitempty) are kept from the existing object.
		// Containers and volumes are merged by name via Kubernetes patch
		// strategy tags, preserving per-element server defaults like
		// terminationMessagePath, defaultMode, etc.
		existingJSON, err := json.Marshal(statefulset.Spec)
		if err != nil {
			return fmt.Errorf("marshal existing StatefulSet spec: %w", err)
		}
		desiredJSON, err := json.Marshal(s.statefulset.Spec)
		if err != nil {
			return fmt.Errorf("marshal desired StatefulSet spec: %w", err)
		}
		patchedJSON, err := strategicpatch.StrategicMergePatch(existingJSON, desiredJSON, appsv1.StatefulSetSpec{})
		if err != nil {
			return fmt.Errorf("strategic merge StatefulSet spec: %w", err)
		}
		if err := json.Unmarshal(patchedJSON, &statefulset.Spec); err != nil {
			return fmt.Errorf("unmarshal patched StatefulSet spec: %w", err)
		}

		err = controllerutil.SetControllerReference(h.GetBeforeObject(), statefulset, h.GetScheme())
		if err != nil {
			return err
		}

		afterU, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(statefulset.DeepCopy())
		if !reflect.DeepEqual(beforeU, afterU) {
			diffFields := findUnstructuredDiff("", beforeU, afterU)
			log.Info("DEBUG unstructured diff inside mutate", "statefulset", s.statefulset.Name, "diffs", diffFields)
		}

		return nil
	})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			h.GetLogger().Info(fmt.Sprintf("StatefulSet %s not found, reconcile in %s", statefulset.Name, s.timeout))
			return ctrl.Result{RequeueAfter: s.timeout}, nil
		}
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		h.GetLogger().Info(fmt.Sprintf("StatefulSet %s - %s", statefulset.Name, op))
	}

	if op == controllerutil.OperationResultNone {
		// Re-read from cache to pick up status updates from the
		// StatefulSet controller (e.g. updated ReadyReplicas).
		s.statefulset, err = GetStatefulSetWithName(ctx, h, statefulset.GetName(), statefulset.GetNamespace())
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// After a create/update the informer cache may still hold the
		// previous version where Generation == ObservedGeneration. Using
		// the server-returned object preserves the correct (bumped)
		// Generation so that callers' readiness checks do not pass on
		// stale data.
		s.statefulset = statefulset
	}

	return ctrl.Result{}, nil
}

// GetStatefulSet - get the statefulset object.
func (s *StatefulSet) GetStatefulSet() appsv1.StatefulSet {
	return *s.statefulset
}

// GetStatefulSetWithName func
func GetStatefulSetWithName(
	ctx context.Context,
	h *helper.Helper,
	name string,
	namespace string,
) (*appsv1.StatefulSet, error) {

	depl := &appsv1.StatefulSet{}
	err := h.GetClient().Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, depl)
	if err != nil {
		return depl, err
	}

	return depl, nil
}

// Delete - delete a statefulset.
func (s *StatefulSet) Delete(
	ctx context.Context,
	h *helper.Helper,
) error {
	err := h.GetClient().Delete(ctx, s.statefulset)
	if err != nil && !k8s_errors.IsNotFound(err) {
		err = fmt.Errorf("error deleting statefulset %s: %w", s.statefulset.Name, err)
		return err
	}

	return nil
}

func findUnstructuredDiff(prefix string, a, b interface{}) []string {
	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			return []string{fmt.Sprintf("TYPE %s: map vs %T", prefix, b)}
		}
		var diffs []string
		allKeys := map[string]bool{}
		for k := range av {
			allKeys[k] = true
		}
		for k := range bv {
			allKeys[k] = true
		}
		for k := range allKeys {
			path := prefix + "." + k
			va, oka := av[k]
			vb, okb := bv[k]
			if !oka {
				diffs = append(diffs, fmt.Sprintf("ADDED %s: %v", path, vb))
			} else if !okb {
				diffs = append(diffs, fmt.Sprintf("REMOVED %s: %v", path, va))
			} else if !reflect.DeepEqual(va, vb) {
				diffs = append(diffs, findUnstructuredDiff(path, va, vb)...)
			}
		}
		return diffs
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok {
			return []string{fmt.Sprintf("TYPE %s: slice vs %T", prefix, b)}
		}
		if len(av) != len(bv) {
			return []string{fmt.Sprintf("LEN %s: %d -> %d", prefix, len(av), len(bv))}
		}
		var diffs []string
		for i := range av {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			if !reflect.DeepEqual(av[i], bv[i]) {
				diffs = append(diffs, findUnstructuredDiff(path, av[i], bv[i])...)
			}
		}
		return diffs
	default:
		if !reflect.DeepEqual(a, b) {
			return []string{fmt.Sprintf("CHANGED %s: %v -> %v", prefix, a, b)}
		}
		return nil
	}
}

// IsReady - validates when deployment is ready deployed to whats being requested
// - the requested replicas in the spec matches the ReadyReplicas of the status
// - all pods run the current spec (UpdatedReplicas == requested replicas)
// - both when the Generatation of the object matches the ObservedGeneration of the Status
// - the rollout is complete (CurrentRevision == UpdateRevision)
func IsReady(deployment appsv1.StatefulSet) bool {
	return deployment.Spec.Replicas != nil &&
		*deployment.Spec.Replicas == deployment.Status.ReadyReplicas &&
		*deployment.Spec.Replicas == deployment.Status.UpdatedReplicas &&
		deployment.Generation == deployment.Status.ObservedGeneration &&
		deployment.Status.CurrentRevision == deployment.Status.UpdateRevision
}
