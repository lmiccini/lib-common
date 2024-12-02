/*
Copyright 2023 Red Hat

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

package clusterrole

import (
	"context"
	"fmt"
	"time"

	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewClusterRole returns an initialized ClusterRole
func NewClusterRole(
	clusterrole *rbacv1.ClusterRole,
	timeout time.Duration,
) *ClusterRole {
	return &ClusterRole{
		clusterrole: clusterrole,
		timeout:     timeout,
	}
}

// CreateOrPatch - creates or patches a role, reconciles after Xs if object won't exist.
func (r *ClusterRole) CreateOrPatch(
	ctx context.Context,
	h *helper.Helper,
) (ctrl.Result, error) {
	clusterrole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.clusterrole.Name,
			Namespace: r.clusterrole.Namespace,
		},
	}

	op, err := controllerutil.CreateOrPatch(ctx, h.GetClient(), clusterrole, func() error {
		clusterrole.Labels = util.MergeStringMaps(clusterrole.Labels, r.clusterrole.Labels)
		clusterrole.Annotations = util.MergeStringMaps(clusterrole.Labels, r.clusterrole.Annotations)
		clusterrole.Rules = r.clusterrole.Rules
		err := controllerutil.SetControllerReference(h.GetBeforeObject(), clusterrole, h.GetScheme())
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			h.GetLogger().Info(fmt.Sprintf("ClusterRole %s not found, reconcile in %s", clusterrole.Name, r.timeout))
			return ctrl.Result{RequeueAfter: r.timeout}, nil
		}
		return ctrl.Result{}, util.WrapErrorForObject(
			fmt.Sprintf("Error creating clusterrole %s", clusterrole.Name),
			clusterrole,
			err,
		)
	}
	if op != controllerutil.OperationResultNone {
		h.GetLogger().Info(fmt.Sprintf("ClusterRole %s - %s", clusterrole.Name, op))
	}

	return ctrl.Result{}, nil
}

// Delete - delete a role
func (r *ClusterRole) Delete(
	ctx context.Context,
	h *helper.Helper,
) error {

	err := h.GetClient().Delete(ctx, r.clusterrole)
	if err != nil && !k8s_errors.IsNotFound(err) {
		err = fmt.Errorf("Error deleting clusterrole %s: %w", r.clusterrole.Name, err)
		return err
	}

	return nil
}
