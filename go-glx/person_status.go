// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package glx

import (
	"strings"
)

// PersonStatus returns the person's `status` property, normalized to lower case
// with surrounding space removed, or "" when the person is nil or has none.
func PersonStatus(person *Person) string {
	if person == nil {
		return ""
	}

	value := person.Properties[PersonPropertyStatus]
	if structured, ok := value.(map[string]any); ok {
		value = structured["value"]
	}

	return strings.ToLower(strings.TrimSpace(propertyText(value)))
}

// IsUnresolvedPlaceholder reports whether the person stands in for someone not
// yet identified: a `hypothetical` or `predicted` status. A `resolved` person,
// or one with a custom status or none, is an ordinary person.
func IsUnresolvedPlaceholder(person *Person) bool {
	switch PersonStatus(person) {
	case PersonStatusHypothetical, PersonStatusPredicted:
		return true
	default:
		return false
	}
}
