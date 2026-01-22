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

package webhook

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateDeprecatedFieldConflict(t *testing.T) {
	deprecatedPath := field.NewPath("spec", "rabbitmqClusterName")
	newPath := field.NewPath("spec", "rabbitmq", "clusterRef")

	tests := []struct {
		name            string
		deprecatedValue string
		newValue        string
		allowBothIfSame bool
		wantWarning     bool
		wantErr         bool
	}{
		{
			name:            "both empty - valid",
			deprecatedValue: "",
			newValue:        "",
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         false,
		},
		{
			name:            "only deprecated set - warning",
			deprecatedValue: "cluster-1",
			newValue:        "",
			allowBothIfSame: true,
			wantWarning:     true,
			wantErr:         false,
		},
		{
			name:            "only new set - valid",
			deprecatedValue: "",
			newValue:        "cluster-1",
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         false,
		},
		{
			name:            "both set with same value, allowBothIfSame=true - warning",
			deprecatedValue: "cluster-1",
			newValue:        "cluster-1",
			allowBothIfSame: true,
			wantWarning:     true,
			wantErr:         false,
		},
		{
			name:            "both set with same value, allowBothIfSame=false - error",
			deprecatedValue: "cluster-1",
			newValue:        "cluster-1",
			allowBothIfSame: false,
			wantWarning:     false,
			wantErr:         true,
		},
		{
			name:            "both set with different values, allowBothIfSame=true - error",
			deprecatedValue: "cluster-1",
			newValue:        "cluster-2",
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         true,
		},
		{
			name:            "both set with different values, allowBothIfSame=false - error",
			deprecatedValue: "cluster-1",
			newValue:        "cluster-2",
			allowBothIfSame: false,
			wantWarning:     false,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning, err := ValidateDeprecatedFieldConflict(
				tt.deprecatedValue,
				tt.newValue,
				deprecatedPath,
				newPath,
				tt.allowBothIfSame,
			)

			if (warning != "") != tt.wantWarning {
				t.Errorf("ValidateDeprecatedFieldConflict() warning = %q, wantWarning %v", warning, tt.wantWarning)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeprecatedFieldConflict() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && err.Type != field.ErrorTypeInvalid {
				t.Errorf("ValidateDeprecatedFieldConflict() error type = %v, want %v", err.Type, field.ErrorTypeInvalid)
			}
		})
	}
}

func TestValidateDeprecatedFieldChange(t *testing.T) {
	deprecatedPath := field.NewPath("spec", "rabbitmqClusterName")
	newPath := field.NewPath("spec", "rabbitmq", "clusterRef")

	tests := []struct {
		name     string
		oldValue string
		newValue string
		wantErr  bool
	}{
		{
			name:     "no change - valid",
			oldValue: "cluster-1",
			newValue: "cluster-1",
			wantErr:  false,
		},
		{
			name:     "clearing field - valid",
			oldValue: "cluster-1",
			newValue: "",
			wantErr:  false,
		},
		{
			name:     "both empty - valid",
			oldValue: "",
			newValue: "",
			wantErr:  false,
		},
		{
			name:     "changing to different value - invalid",
			oldValue: "cluster-1",
			newValue: "cluster-2",
			wantErr:  true,
		},
		{
			name:     "setting from empty - invalid",
			oldValue: "",
			newValue: "cluster-1",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeprecatedFieldChange(
				tt.oldValue,
				tt.newValue,
				deprecatedPath,
				newPath,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeprecatedFieldChange() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && err.Type != field.ErrorTypeForbidden {
				t.Errorf("ValidateDeprecatedFieldChange() error type = %v, want %v", err.Type, field.ErrorTypeForbidden)
			}
		})
	}
}

func TestValidateDeprecatedFieldConflictPtr(t *testing.T) {
	deprecatedPath := field.NewPath("spec", "rabbitmqNotificationsBus")
	newPath := field.NewPath("spec", "rabbitmq", "notificationsBus")

	// Helper to create string pointers
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name            string
		deprecatedValue *string
		newValue        *string
		allowBothIfSame bool
		wantWarning     bool
		wantErr         bool
	}{
		{
			name:            "both nil - valid",
			deprecatedValue: nil,
			newValue:        nil,
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         false,
		},
		{
			name:            "only deprecated set - warning",
			deprecatedValue: strPtr("bus-1"),
			newValue:        nil,
			allowBothIfSame: true,
			wantWarning:     true,
			wantErr:         false,
		},
		{
			name:            "only new set - valid",
			deprecatedValue: nil,
			newValue:        strPtr("bus-1"),
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         false,
		},
		{
			name:            "both set with same value, allowBothIfSame=true - warning",
			deprecatedValue: strPtr("bus-1"),
			newValue:        strPtr("bus-1"),
			allowBothIfSame: true,
			wantWarning:     true,
			wantErr:         false,
		},
		{
			name:            "both set with same value, allowBothIfSame=false - error",
			deprecatedValue: strPtr("bus-1"),
			newValue:        strPtr("bus-1"),
			allowBothIfSame: false,
			wantWarning:     false,
			wantErr:         true,
		},
		{
			name:            "both set with different values - error",
			deprecatedValue: strPtr("bus-1"),
			newValue:        strPtr("bus-2"),
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         true,
		},
		{
			name:            "deprecated empty string, new nil - valid",
			deprecatedValue: strPtr(""),
			newValue:        nil,
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         false,
		},
		{
			name:            "deprecated nil, new empty string - valid",
			deprecatedValue: nil,
			newValue:        strPtr(""),
			allowBothIfSame: true,
			wantWarning:     false,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning, err := ValidateDeprecatedFieldConflictPtr(
				tt.deprecatedValue,
				tt.newValue,
				deprecatedPath,
				newPath,
				tt.allowBothIfSame,
			)

			if (warning != "") != tt.wantWarning {
				t.Errorf("ValidateDeprecatedFieldConflictPtr() warning = %q, wantWarning %v", warning, tt.wantWarning)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeprecatedFieldConflictPtr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDeprecatedFieldChangePtr(t *testing.T) {
	deprecatedPath := field.NewPath("spec", "rabbitmqNotificationsBus")
	newPath := field.NewPath("spec", "rabbitmq", "notificationsBus")

	// Helper to create string pointers
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		oldValue *string
		newValue *string
		wantErr  bool
	}{
		{
			name:     "no change - both nil - valid",
			oldValue: nil,
			newValue: nil,
			wantErr:  false,
		},
		{
			name:     "no change - both same value - valid",
			oldValue: strPtr("bus-1"),
			newValue: strPtr("bus-1"),
			wantErr:  false,
		},
		{
			name:     "clearing field - valid",
			oldValue: strPtr("bus-1"),
			newValue: nil,
			wantErr:  false,
		},
		{
			name:     "clearing to empty string - valid",
			oldValue: strPtr("bus-1"),
			newValue: strPtr(""),
			wantErr:  false,
		},
		{
			name:     "changing to different value - invalid",
			oldValue: strPtr("bus-1"),
			newValue: strPtr("bus-2"),
			wantErr:  true,
		},
		{
			name:     "setting from nil - invalid",
			oldValue: nil,
			newValue: strPtr("bus-1"),
			wantErr:  true,
		},
		{
			name:     "setting from empty string - invalid",
			oldValue: strPtr(""),
			newValue: strPtr("bus-1"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeprecatedFieldChangePtr(
				tt.oldValue,
				tt.newValue,
				deprecatedPath,
				newPath,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeprecatedFieldChangePtr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test structs for reflection-based validation
type TestSpec struct {
	// Deprecated: Use MessagingBus.Cluster instead
	RabbitMqClusterName string `json:"rabbitMqClusterName,omitempty" deprecated:"messagingBus.cluster"`

	// Deprecated: Use NotificationsBus.Cluster instead
	NotificationsBusInstance *string `json:"notificationsBusInstance,omitempty" deprecated:"notificationsBus.cluster"`

	MessagingBus     TestMessagingBus  `json:"messagingBus,omitempty"`
	NotificationsBus *TestMessagingBus `json:"notificationsBus,omitempty"`
}

type TestMessagingBus struct {
	Cluster string `json:"cluster,omitempty"`
}

func TestValidateDeprecatedFieldsCreate(t *testing.T) {
	basePath := field.NewPath("spec")

	tests := []struct {
		name         string
		spec         TestSpec
		wantWarnings int
	}{
		{
			name: "no deprecated fields set",
			spec: TestSpec{
				MessagingBus: TestMessagingBus{Cluster: "cluster-1"},
			},
			wantWarnings: 0,
		},
		{
			name: "only deprecated string field set",
			spec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			wantWarnings: 1,
		},
		{
			name: "deprecated and new string field set to same value",
			spec: TestSpec{
				RabbitMqClusterName: "cluster-1",
				MessagingBus:        TestMessagingBus{Cluster: "cluster-1"},
			},
			wantWarnings: 1,
		},
		{
			name: "only deprecated pointer field set",
			spec: TestSpec{
				NotificationsBusInstance: func() *string { s := "bus-1"; return &s }(),
			},
			wantWarnings: 1,
		},
		{
			name: "deprecated and new pointer field set to same value",
			spec: TestSpec{
				NotificationsBusInstance: func() *string { s := "bus-1"; return &s }(),
				NotificationsBus:         &TestMessagingBus{Cluster: "bus-1"},
			},
			wantWarnings: 1,
		},
		{
			name: "both deprecated fields set",
			spec: TestSpec{
				RabbitMqClusterName:      "cluster-1",
				NotificationsBusInstance: func() *string { s := "bus-1"; return &s }(),
			},
			wantWarnings: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the deprecated fields slice from the spec
			deprecatedFields := []DeprecatedField{
				{
					DeprecatedFieldName: "rabbitMqClusterName",
					NewFieldPath:        []string{"messagingBus", "cluster"},
					DeprecatedValue:     &tt.spec.RabbitMqClusterName,
					NewValue:            &tt.spec.MessagingBus.Cluster,
				},
				{
					DeprecatedFieldName: "notificationsBusInstance",
					NewFieldPath:        []string{"notificationsBus", "cluster"},
					DeprecatedValue:     tt.spec.NotificationsBusInstance,
					NewValue: func() *string {
						if tt.spec.NotificationsBus != nil {
							return &tt.spec.NotificationsBus.Cluster
						}
						return nil
					}(),
				},
			}
			warnings := ValidateDeprecatedFieldsCreate(deprecatedFields, basePath)
			if len(warnings) != tt.wantWarnings {
				t.Errorf("ValidateDeprecatedFieldsCreate() got %d warnings, want %d. Warnings: %v",
					len(warnings), tt.wantWarnings, warnings)
			}
		})
	}
}

func TestValidateDeprecatedFieldsUpdate(t *testing.T) {
	basePath := field.NewPath("spec")

	tests := []struct {
		name         string
		oldSpec      TestSpec
		newSpec      TestSpec
		wantWarnings int
		wantErrors   int
	}{
		{
			name: "no changes",
			oldSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			newSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			wantWarnings: 1, // Warning for using deprecated field
			wantErrors:   0,
		},
		{
			name: "clearing deprecated field",
			oldSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			newSpec: TestSpec{
				RabbitMqClusterName: "",
				MessagingBus:        TestMessagingBus{Cluster: "cluster-1"},
			},
			wantWarnings: 0,
			wantErrors:   0,
		},
		{
			name: "changing deprecated field - invalid",
			oldSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			newSpec: TestSpec{
				RabbitMqClusterName: "cluster-2",
			},
			wantWarnings: 1, // Warning for using deprecated field
			wantErrors:   1, // Error for changing it
		},
		{
			name: "setting from empty - invalid",
			oldSpec: TestSpec{
				RabbitMqClusterName: "",
			},
			newSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			wantWarnings: 1,
			wantErrors:   1,
		},
		{
			name: "both deprecated and new set with different values - error",
			oldSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
			},
			newSpec: TestSpec{
				RabbitMqClusterName: "cluster-1",
				MessagingBus:        TestMessagingBus{Cluster: "cluster-2"},
			},
			wantWarnings: 0,
			wantErrors:   1,
		},
		{
			name: "pointer field - clearing",
			oldSpec: TestSpec{
				NotificationsBusInstance: func() *string { s := "bus-1"; return &s }(),
			},
			newSpec: TestSpec{
				NotificationsBusInstance: nil,
				NotificationsBus:         &TestMessagingBus{Cluster: "bus-1"},
			},
			wantWarnings: 0,
			wantErrors:   0,
		},
		{
			name: "pointer field - changing - invalid",
			oldSpec: TestSpec{
				NotificationsBusInstance: func() *string { s := "bus-1"; return &s }(),
			},
			newSpec: TestSpec{
				NotificationsBusInstance: func() *string { s := "bus-2"; return &s }(),
			},
			wantWarnings: 1,
			wantErrors:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the deprecated fields update slice from the specs
			deprecatedFields := []DeprecatedFieldUpdate{
				{
					DeprecatedFieldName: "rabbitMqClusterName",
					NewFieldPath:        []string{"messagingBus", "cluster"},
					OldDeprecatedValue:  &tt.oldSpec.RabbitMqClusterName,
					NewDeprecatedValue:  &tt.newSpec.RabbitMqClusterName,
					NewValue:            &tt.newSpec.MessagingBus.Cluster,
				},
				{
					DeprecatedFieldName: "notificationsBusInstance",
					NewFieldPath:        []string{"notificationsBus", "cluster"},
					OldDeprecatedValue:  tt.oldSpec.NotificationsBusInstance,
					NewDeprecatedValue:  tt.newSpec.NotificationsBusInstance,
					NewValue: func() *string {
						if tt.newSpec.NotificationsBus != nil {
							return &tt.newSpec.NotificationsBus.Cluster
						}
						return nil
					}(),
				},
			}
			warnings, errors := ValidateDeprecatedFieldsUpdate(deprecatedFields, basePath)
			if len(warnings) != tt.wantWarnings {
				t.Errorf("ValidateDeprecatedFieldsUpdate() got %d warnings, want %d. Warnings: %v",
					len(warnings), tt.wantWarnings, warnings)
			}
			if len(errors) != tt.wantErrors {
				t.Errorf("ValidateDeprecatedFieldsUpdate() got %d errors, want %d. Errors: %v",
					len(errors), tt.wantErrors, errors)
			}
		})
	}
}
