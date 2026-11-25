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
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	vocabularies "github.com/genealogix/glx/specification/5-standard-vocabularies"
)

// Every vocabulary map on GLXFile is in vocabFields or propertyFields, under
// its own YAML key, and names a standard vocabulary file that exists.
func TestVocabFieldsCoverGLXFile(t *testing.T) {
	var g GLXFile
	listed := make(map[uintptr]string)
	for _, f := range g.vocabFields() {
		listed[reflect.ValueOf(f.Map).Pointer()] = f.Key
		assert.Contains(t, vocabularies.Files, f.Filename(), f.Key)
	}
	for _, f := range g.propertyFields() {
		listed[reflect.ValueOf(f.Map).Pointer()] = f.Key
		assert.Contains(t, vocabularies.Files, f.Filename(), f.Key)
	}

	vocabType := reflect.TypeFor[map[string]*VocabularyEntry]()
	propType := reflect.TypeFor[map[string]*PropertyDefinition]()
	v := reflect.ValueOf(&g).Elem()
	want := 0
	for i := range v.NumField() {
		field := v.Type().Field(i)
		if field.Type != vocabType && field.Type != propType {
			continue
		}
		want++
		key, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		assert.Equal(t, key, listed[v.Field(i).Addr().Pointer()],
			"GLXFile.%s must be listed in vocabFields or propertyFields under its YAML key", field.Name)
	}
	assert.Len(t, listed, want, "every listed vocabulary is a distinct GLXFile field")
}
