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

package statefulset

import (
	corev1 "k8s.io/api/core/v1"
)

// MergeVolumesByName merges desired volumes into existing volumes matched by
// name. For each match it preserves server-defaulted fields (defaultMode for
// Secret/ConfigMap/Projected/DownwardAPI volumes, type for HostPath volumes)
// from the existing volume when the desired volume doesn't set them.
//
// When volume counts differ or a desired volume name is not found in existing,
// the existing slice is replaced with the desired volumes.
func MergeVolumesByName(existing *[]corev1.Volume, desired []corev1.Volume) {
	if len(*existing) != len(desired) {
		*existing = desired
		return
	}

	existingByName := make(map[string]int, len(*existing))
	for i := range *existing {
		existingByName[(*existing)[i].Name] = i
	}

	for _, d := range desired {
		idx, ok := existingByName[d.Name]
		if !ok {
			*existing = desired
			return
		}
		e := (*existing)[idx]

		if d.Secret != nil && e.Secret != nil && d.Secret.DefaultMode == nil {
			d.Secret.DefaultMode = e.Secret.DefaultMode
		}
		if d.ConfigMap != nil && e.ConfigMap != nil && d.ConfigMap.DefaultMode == nil {
			d.ConfigMap.DefaultMode = e.ConfigMap.DefaultMode
		}
		if d.Projected != nil && e.Projected != nil && d.Projected.DefaultMode == nil {
			d.Projected.DefaultMode = e.Projected.DefaultMode
		}
		if d.DownwardAPI != nil && e.DownwardAPI != nil && d.DownwardAPI.DefaultMode == nil {
			d.DownwardAPI.DefaultMode = e.DownwardAPI.DefaultMode
		}
		if d.HostPath != nil && e.HostPath != nil && d.HostPath.Type == nil {
			d.HostPath.Type = e.HostPath.Type
		}

		(*existing)[idx] = d
	}
}
