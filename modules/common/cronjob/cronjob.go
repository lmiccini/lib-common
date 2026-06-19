/*
Copyright 2022 Red Hat

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

// Package cronjob provides utilities for managing Kubernetes CronJob resources
package cronjob

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/pod"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewCronJob returns an initialized CronJob.
func NewCronJob(
	cronjob *batchv1.CronJob,
	timeout time.Duration,
) *CronJob {
	return &CronJob{
		cronjob: cronjob,
		timeout: timeout,
	}
}

// CreateOrPatch - creates or patches a cronjob, reconciles after Xs if object won't exist.
func (cj *CronJob) CreateOrPatch(
	ctx context.Context,
	h *helper.Helper,
) (ctrl.Result, error) {
	cronjob := &batchv1.CronJob{}
	cronjob.ObjectMeta = cj.cronjob.ObjectMeta

	log := ctrl.Log.WithName("cronjob-debug")
	op, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), cronjob, func() error {
		beforeU, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(cronjob.DeepCopy())

		pod.SetPullPolicyDefaults(&cj.cronjob.Spec.JobTemplate.Spec.Template.Spec)

		existingJSON, err := json.Marshal(cronjob.Spec)
		if err != nil {
			return fmt.Errorf("marshal existing CronJob spec: %w", err)
		}
		desiredJSON, err := json.Marshal(cj.cronjob.Spec)
		if err != nil {
			return fmt.Errorf("marshal desired CronJob spec: %w", err)
		}
		patchedJSON, err := strategicpatch.StrategicMergePatch(existingJSON, desiredJSON, batchv1.CronJobSpec{})
		if err != nil {
			return fmt.Errorf("strategic merge CronJob spec: %w", err)
		}
		if err := json.Unmarshal(patchedJSON, &cronjob.Spec); err != nil {
			return fmt.Errorf("unmarshal patched CronJob spec: %w", err)
		}

		err = controllerutil.SetControllerReference(h.GetBeforeObject(), cronjob, h.GetScheme())
		if err != nil {
			return err
		}

		afterU, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(cronjob.DeepCopy())
		if !reflect.DeepEqual(beforeU, afterU) {
			diffFields := findCronJobDiff("", beforeU, afterU)
			log.Info("DEBUG unstructured diff inside mutate", "cronjob", cj.cronjob.Name, "diffs", diffFields)
		}

		return nil
	})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			h.GetLogger().Info(fmt.Sprintf("CronJob %s not found, reconcile in %s", cj.cronjob.Name, cj.timeout))
			return ctrl.Result{RequeueAfter: cj.timeout}, nil
		}
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		h.GetLogger().Info(fmt.Sprintf("CronJob %s - %s", cj.cronjob.Name, op))
	}

	return ctrl.Result{}, nil
}

// Delete - delete a cronjob.
func (cj *CronJob) Delete(
	ctx context.Context,
	h *helper.Helper,
) error {
	err := h.GetClient().Delete(ctx, cj.cronjob)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return fmt.Errorf("error deleting cronjob %s: %w", cj.cronjob.Name, err)
	}

	return nil
}

// GetCronJob - get the cronjob object.
func (cj *CronJob) GetCronJob() batchv1.CronJob {
	return *cj.cronjob
}

// GetCronJobWithName func
func GetCronJobWithName(
	ctx context.Context,
	h *helper.Helper,
	name string,
	namespace string,
) (*batchv1.CronJob, error) {

	cronjob := &batchv1.CronJob{}
	err := h.GetClient().Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cronjob)
	if err != nil {
		return cronjob, err
	}

	return cronjob, nil
}

// GetCronJobByName - returns a *CronJob object with specified name and namespace
func GetCronJobByName(
	ctx context.Context,
	h *helper.Helper,
	name string,
	namespace string,
) (*CronJob, error) {
	cronjob, err := GetCronJobWithName(ctx, h, name, namespace)
	cj := &CronJob{
		cronjob: cronjob,
	}
	if err != nil {
		return cj, err
	}

	return cj, nil
}

func findCronJobDiff(prefix string, a, b interface{}) []string {
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
				diffs = append(diffs, findCronJobDiff(path, va, vb)...)
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
				diffs = append(diffs, findCronJobDiff(path, av[i], bv[i])...)
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
