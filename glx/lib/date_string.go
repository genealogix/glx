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

package lib

import "gopkg.in/yaml.v3"

// DateString is a string type that always marshals to YAML with quotes.
// This ensures consistent formatting of dates in GLX files, regardless of
// whether they contain qualifiers (ABT, BEF, AFT), ranges (BET...AND), or
// simple ISO 8601 dates.
//
// Examples:
//   - "1850"
//   - "ABT 1850"
//   - "BEF 1920-01-15"
//   - "BET 1880 AND 1890"
//   - "1635-11"
type DateString string

// MarshalYAML implements yaml.Marshaler to ensure dates are always quoted.
func (ds DateString) MarshalYAML() (any, error) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.DoubleQuotedStyle,
		Value: string(ds),
	}

	return node, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for DateString.
func (ds *DateString) UnmarshalYAML(node *yaml.Node) error {
	*ds = DateString(node.Value)

	return nil
}

// String returns the underlying string value.
func (ds DateString) String() string {
	return string(ds)
}
