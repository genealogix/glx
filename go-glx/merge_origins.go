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
	"fmt"
	"reflect"
	"strings"
)

// keyOrigins remembers which file first defined each key of each top-level
// collection while DeserializeMultiFileFromMap merges files, so a duplicate
// ID can be reported with both paths instead of the ID alone (#1232).
type keyOrigins map[string]map[string]string

func newKeyOrigins() keyOrigins { return keyOrigins{} }

// record notes filePath as the origin of every key in partial that has no
// origin yet. The first file to define a key is the one Merge keeps.
func (o keyOrigins) record(partial *GLXFile, filePath string) {
	v := reflect.ValueOf(partial).Elem()
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() || field.Type.Kind() != reflect.Map {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			continue
		}
		m := v.Field(i)
		if m.Len() == 0 {
			continue
		}
		if o[name] == nil {
			o[name] = map[string]string{}
		}
		for _, key := range m.MapKeys() {
			id := key.String()
			if _, seen := o[name][id]; !seen {
				o[name][id] = filePath
			}
		}
	}
}

// annotate appends "(<first file> vs <filePath>)" to every Merge conflict of
// the form "conflict <collection> ID: <id>" whose first definition is known.
// Any other conflict text is returned unchanged.
func (o keyOrigins) annotate(conflicts []string, filePath string) []string {
	for i, c := range conflicts {
		rest, ok := strings.CutPrefix(c, "conflict ")
		if !ok {
			continue
		}
		collection, id, ok := strings.Cut(rest, " ID: ")
		if !ok {
			continue
		}
		if first, ok := o[collection][id]; ok {
			conflicts[i] = fmt.Sprintf("%s (%s vs %s)", c, first, filePath)
		}
	}

	return conflicts
}
