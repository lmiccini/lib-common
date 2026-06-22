/*
Copyright 2026 Red Hat

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

// Package subcr provides shared helpers for parent controllers that own
// service sub-CRs and must guard credential rotation.
package subcr

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// StabilityTracker records whether any sub-CR CreateOrPatch changed the spec
// during the current parent reconcile pass. Track one instance per reconcile
// and call Record after each sub-CR create/update.
type StabilityTracker struct {
	stable bool
}

// NewStabilityTracker returns a tracker that starts in the stable state.
func NewStabilityTracker() *StabilityTracker {
	return &StabilityTracker{stable: true}
}

// Record updates the tracker from a CreateOrPatch operation result.
func (t *StabilityTracker) Record(op controllerutil.OperationResult) {
	if op != controllerutil.OperationResultNone {
		t.stable = false
	}
}

// Stable reports whether every tracked sub-CR CreateOrPatch returned
// OperationResultNone in this reconcile pass.
func (t *StabilityTracker) Stable() bool {
	return t.stable
}
