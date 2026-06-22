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

package object

import (
	"context"
	"testing"

	. "github.com/onsi/gomega" // nolint:revive

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

var (
	metadata = metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion:         "core.openstack.org/v1beta1",
				BlockOwnerDeletion: ptr.To(true),
				Controller:         ptr.To(true),
				Kind:               "OpenStackControlPlane",
				Name:               "openstack-network-isolation",
				UID:                "11111111-1111-1111-1111-111111111111",
			},
		},
	}
)

func TestCheckOwnerRefExist(t *testing.T) {
	tests := []struct {
		name      string
		ownerRefs []metav1.OwnerReference
		uid       types.UID
		want      bool
	}{
		{
			name:      "Check existing owner",
			ownerRefs: metadata.OwnerReferences,
			uid:       types.UID("11111111-1111-1111-1111-111111111111"),
			want:      true,
		},
		{
			name:      "Check non existing owner",
			ownerRefs: metadata.OwnerReferences,
			uid:       types.UID("22222222-2222-2222-2222-222222222222"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			g.Expect(CheckOwnerRefExist(tt.uid, tt.ownerRefs)).To(BeIdenticalTo(tt.want))
		})
	}
}

func TestFinalizeSecretRotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name          string
		statusSecret  string
		currentSecret string
		guardReady    bool
		wantSecret    string
		wantErr       bool
	}{
		{
			name:          "no rotation - empty status",
			statusSecret:  "",
			currentSecret: "new-secret",
			guardReady:    true,
			wantSecret:    "new-secret",
		},
		{
			name:          "no rotation - same secret",
			statusSecret:  "same-secret",
			currentSecret: "same-secret",
			guardReady:    true,
			wantSecret:    "same-secret",
		},
		{
			name:          "rotation in progress - guard not ready",
			statusSecret:  "old-secret",
			currentSecret: "new-secret",
			guardReady:    false,
			wantSecret:    "old-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			got, err := FinalizeSecretRotation(
				ctx, nil, "openstack",
				tt.statusSecret, tt.currentSecret,
				"openstack.org/test-consumer",
				tt.guardReady,
			)

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(got).To(Equal(tt.wantSecret))
		})
	}
}
