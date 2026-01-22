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

// Package webhook provides validation utilities for admission webhooks
package webhook

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateDeprecatedFieldConflict validates that deprecated and new fields are not set
// in conflicting ways. This is used during creation or update when transitioning from
// a deprecated field to a new field.
//
// Parameters:
//   - deprecatedValue: Current value of the deprecated field
//   - newValue: Current value of the new field
//   - deprecatedFieldPath: Path to the deprecated field (for error messages)
//   - newFieldPath: Path to the new field (for error messages)
//   - allowBothIfSame: If true, allows both fields to be set if they have the same value.
//     This is useful when webhook defaulting copies the deprecated field to the new field.
//     If false, strictly enforces that only one field can be set at a time.
//
// Returns a warning if only the deprecated field is set (to encourage migration).
// Returns an error if both fields are set with different values, or if both are set
// and allowBothIfSame is false.
//
// Example usage:
//
//	warnings, err := webhook.ValidateDeprecatedFieldConflict(
//	    spec.RabbitmqClusterName,           // deprecated field
//	    spec.RabbitMQ.ClusterRef,           // new field
//	    field.NewPath("spec", "rabbitmqClusterName"),
//	    field.NewPath("spec", "rabbitmq", "clusterRef"),
//	    true,                               // allow both if same (for webhook defaulting)
//	)
func ValidateDeprecatedFieldConflict(
	deprecatedValue, newValue string,
	deprecatedFieldPath, newFieldPath *field.Path,
	allowBothIfSame bool,
) (string, *field.Error) {
	// Allow if both are empty
	if deprecatedValue == "" && newValue == "" {
		return "", nil
	}

	// Only deprecated field is set - return warning
	if deprecatedValue != "" && newValue == "" {
		warning := fmt.Sprintf("field %q is deprecated, please use %q instead",
			deprecatedFieldPath.String(),
			newFieldPath.String(),
		)
		return warning, nil
	}

	// Only new field is set - this is the desired state
	if deprecatedValue == "" && newValue != "" {
		return "", nil
	}

	// Both fields are set - check if they have the same value
	if deprecatedValue == newValue {
		if allowBothIfSame {
			// Allow but warn - this supports webhook defaulting patterns
			warning := fmt.Sprintf("both %q and %q are set with the same value. Please migrate to using only %q and clear %q",
				deprecatedFieldPath.String(),
				newFieldPath.String(),
				newFieldPath.String(),
				deprecatedFieldPath.String(),
			)
			return warning, nil
		}
		// Strict mode - reject even if same
		return "", field.Invalid(
			deprecatedFieldPath,
			deprecatedValue,
			fmt.Sprintf("cannot set both deprecated field %q and new field %q. Use %q only",
				deprecatedFieldPath.String(),
				newFieldPath.String(),
				newFieldPath.String(),
			),
		)
	}

	// Both are set with different values - this is always an error
	return "", field.Invalid(
		deprecatedFieldPath,
		deprecatedValue,
		fmt.Sprintf("cannot set both deprecated field %q and new field %q with different values (deprecated: %q, new: %q). Use %q only",
			deprecatedFieldPath.String(),
			newFieldPath.String(),
			deprecatedValue,
			newValue,
			newFieldPath.String(),
		),
	)
}

// ValidateDeprecatedFieldChange prevents modifications to deprecated fields unless being cleared.
// This is used during updates to ensure users migrate to the new field instead of continuing
// to use the deprecated one.
//
// Returns an error if the deprecated field is being changed to a non-empty value.
// Returns nil if:
//   - The field is not being changed
//   - The field is being cleared (set to empty)
//
// Example usage:
//
//	err := webhook.ValidateDeprecatedFieldChange(
//	    old.Spec.RabbitmqClusterName,  // old value
//	    new.Spec.RabbitmqClusterName,  // new value
//	    field.NewPath("spec", "rabbitmqClusterName"),
//	    field.NewPath("spec", "rabbitmq", "clusterRef"),  // suggested new field
//	)
func ValidateDeprecatedFieldChange(
	oldValue, newValue string,
	deprecatedFieldPath, newFieldPath *field.Path,
) *field.Error {
	// Allow if not changing
	if oldValue == newValue {
		return nil
	}

	// Allow if clearing the field (migrating away)
	if newValue == "" {
		return nil
	}

	// Reject changes to non-empty values
	return field.Forbidden(
		deprecatedFieldPath,
		fmt.Sprintf("field %q is deprecated, use %q instead. To migrate, first set %q, then clear this field",
			deprecatedFieldPath.String(),
			newFieldPath.String(),
			newFieldPath.String(),
		),
	)
}

// ValidateDeprecatedFieldConflictPtr is a pointer-safe variant of ValidateDeprecatedFieldConflict.
// It handles *string fields by dereferencing them before validation.
//
// Example usage:
//
//	warnings, err := webhook.ValidateDeprecatedFieldConflictPtr(
//	    spec.RabbitmqNotificationsBus,      // *string deprecated field
//	    spec.RabbitMQ.NotificationsBus,     // *string new field
//	    field.NewPath("spec", "rabbitmqNotificationsBus"),
//	    field.NewPath("spec", "rabbitmq", "notificationsBus"),
//	    true,                               // allow both if same
//	)
func ValidateDeprecatedFieldConflictPtr(
	deprecatedValue, newValue *string,
	deprecatedFieldPath, newFieldPath *field.Path,
	allowBothIfSame bool,
) (string, *field.Error) {
	// Dereference pointers, treating nil as empty string
	oldVal := ""
	if deprecatedValue != nil {
		oldVal = *deprecatedValue
	}

	newVal := ""
	if newValue != nil {
		newVal = *newValue
	}

	return ValidateDeprecatedFieldConflict(oldVal, newVal, deprecatedFieldPath, newFieldPath, allowBothIfSame)
}

// ValidateDeprecatedFieldChangePtr is a pointer-safe variant of ValidateDeprecatedFieldChange.
// It handles *string fields by dereferencing them before validation.
//
// Example usage:
//
//	err := webhook.ValidateDeprecatedFieldChangePtr(
//	    old.Spec.RabbitmqNotificationsBus,  // *string old value
//	    new.Spec.RabbitmqNotificationsBus,  // *string new value
//	    field.NewPath("spec", "rabbitmqNotificationsBus"),
//	    field.NewPath("spec", "rabbitmq", "notificationsBus"),
//	)
func ValidateDeprecatedFieldChangePtr(
	oldValue, newValue *string,
	deprecatedFieldPath, newFieldPath *field.Path,
) *field.Error {
	// Dereference pointers, treating nil as empty string
	oldVal := ""
	if oldValue != nil {
		oldVal = *oldValue
	}

	newVal := ""
	if newValue != nil {
		newVal = *newValue
	}

	return ValidateDeprecatedFieldChange(oldVal, newVal, deprecatedFieldPath, newFieldPath)
}

// DeprecatedField represents a mapping from a deprecated field to its replacement.
type DeprecatedField struct {
	// DeprecatedFieldName is the JSON field name of the deprecated field (e.g., "rabbitMqClusterName")
	DeprecatedFieldName string
	// NewFieldPath is the path to the new field (e.g., []string{"messagingBus", "cluster"})
	NewFieldPath []string
	// DeprecatedValue is the current value of the deprecated field
	DeprecatedValue *string
	// NewValue is the current value of the new field
	NewValue *string
}

// ValidateDeprecatedFieldsCreate validates deprecated fields during CREATE operations.
// Operators should build an explicit list of deprecated field mappings and pass them to this function.
//
// Example usage:
//
//	deprecatedFields := []webhook.DeprecatedField{
//	    {
//	        DeprecatedFieldName: "rabbitMqClusterName",
//	        NewFieldPath:        []string{"messagingBus", "cluster"},
//	        DeprecatedValue:     &spec.RabbitMqClusterName,
//	        NewValue:            &spec.MessagingBus.Cluster,
//	    },
//	    {
//	        DeprecatedFieldName: "notificationsBusInstance",
//	        NewFieldPath:        []string{"notificationsBus", "cluster"},
//	        DeprecatedValue:     spec.NotificationsBusInstance,
//	        NewValue:            spec.NotificationsBus.Cluster,
//	    },
//	}
//	warnings := common_webhook.ValidateDeprecatedFieldsCreate(deprecatedFields, basePath)
func ValidateDeprecatedFieldsCreate(deprecatedFields []DeprecatedField, basePath *field.Path) []string {
	var allWarnings []string

	for _, df := range deprecatedFields {
		deprecatedPath := basePath.Child(df.DeprecatedFieldName)
		newPath := basePath
		for _, segment := range df.NewFieldPath {
			newPath = newPath.Child(segment)
		}

		warning, _ := ValidateDeprecatedFieldConflictPtr(
			df.DeprecatedValue,
			df.NewValue,
			deprecatedPath,
			newPath,
			true, // allowBothIfSame
		)
		if warning != "" {
			allWarnings = append(allWarnings, warning)
		}
	}

	return allWarnings
}

// DeprecatedFieldUpdate represents a mapping from a deprecated field to its replacement during UPDATE.
type DeprecatedFieldUpdate struct {
	// DeprecatedFieldName is the JSON field name of the deprecated field (e.g., "rabbitMqClusterName")
	DeprecatedFieldName string
	// NewFieldPath is the path to the new field (e.g., []string{"messagingBus", "cluster"})
	NewFieldPath []string
	// OldDeprecatedValue is the old value of the deprecated field
	OldDeprecatedValue *string
	// NewDeprecatedValue is the new value of the deprecated field
	NewDeprecatedValue *string
	// NewValue is the current value of the new field
	NewValue *string
}

// ValidateDeprecatedFieldsUpdate validates deprecated fields during UPDATE operations.
// Operators should build an explicit list of deprecated field mappings and pass them to this function.
//
// Example usage:
//
//	deprecatedFields := []webhook.DeprecatedFieldUpdate{
//	    {
//	        DeprecatedFieldName: "rabbitMqClusterName",
//	        NewFieldPath:        []string{"messagingBus", "cluster"},
//	        OldDeprecatedValue:  &old.RabbitMqClusterName,
//	        NewDeprecatedValue:  &new.RabbitMqClusterName,
//	        NewValue:            &new.MessagingBus.Cluster,
//	    },
//	}
//	warnings, errors := common_webhook.ValidateDeprecatedFieldsUpdate(deprecatedFields, basePath)
func ValidateDeprecatedFieldsUpdate(deprecatedFields []DeprecatedFieldUpdate, basePath *field.Path) ([]string, field.ErrorList) {
	var allWarnings []string
	var allErrors field.ErrorList

	for _, df := range deprecatedFields {
		deprecatedPath := basePath.Child(df.DeprecatedFieldName)
		newPath := basePath
		for _, segment := range df.NewFieldPath {
			newPath = newPath.Child(segment)
		}

		// Check for conflicts between deprecated and new field
		warning, err := ValidateDeprecatedFieldConflictPtr(
			df.NewDeprecatedValue,
			df.NewValue,
			deprecatedPath,
			newPath,
			true, // allowBothIfSame
		)
		if warning != "" {
			allWarnings = append(allWarnings, warning)
		}
		if err != nil {
			allErrors = append(allErrors, err)
		}

		// Check for invalid changes to the deprecated field
		if err := ValidateDeprecatedFieldChangePtr(
			df.OldDeprecatedValue,
			df.NewDeprecatedValue,
			deprecatedPath,
			newPath,
		); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return allWarnings, allErrors
}
