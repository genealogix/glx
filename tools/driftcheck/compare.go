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

package main

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"
)

// checker holds the schema set under comparison and accumulates findings. It is
// pure with respect to the filesystem: callers load schemas (and the Go types
// to compare, via reflect) and feed them in.
type checker struct {
	schemas   *schemaSet
	typesFile string // source file findings are attributed to, for the allowlist
	findings  []finding
}

func newChecker(schemas *schemaSet, typesFile string) *checker {
	return &checker{schemas: schemas, typesFile: typesFile}
}

func (c *checker) add(f *finding) {
	c.findings = append(c.findings, *f)
}

// compareStruct compares a Go struct type against an object schema node:
// field/property presence (both directions), omitempty vs required, and (when
// recurse is true) the type of every matched field.
//
// The label argument seeds the owning-type name for report grouping and
// per-symbol allowlist identity, but the actual Go type name takes precedence
// whenever it is available. That distinction matters during recursion: a nested
// type (Participant, EntityRef, Submitter, Search, …) is reported and
// allowlisted under its own Go type (e.g. "Participant.role"), not the parent
// entity it was reached through. For a top-level binding the label already
// equals the type name, so behavior there is unchanged.
func (c *checker) compareStruct(label string, goType reflect.Type, node *schemaNode, loc ref, recurse bool) {
	goType = derefType(goType)
	if goType.Kind() != reflect.Struct {
		c.add(&finding{
			Severity: severityCritical, Entity: label, Category: catInternal,
			File: c.typesFile, Symbol: label,
			Message: fmt.Sprintf("expected struct for %s, got %s", label, goType.Kind()),
		})

		return
	}

	owner := goType.Name()
	if owner == "" {
		owner = label // anonymous struct: fall back to the caller's label
	}

	props := node.Properties
	required := stringSet(node.Required)
	closed := node.additionalPropsClosed()
	goFields := structFieldsByYAML(goType)

	// Direction 1: schema property with no Go field.
	for _, propName := range sortedKeys(props) {
		if _, ok := goFields.m[propName]; ok {
			continue
		}
		sev := severityMajor
		qualifier := "optional"
		if required[propName] {
			sev = severityCritical
			qualifier = "required"
		}
		c.add(&finding{
			Severity: sev, Entity: owner, Category: catFieldPresence,
			File: c.typesFile, Symbol: owner + "." + propName, YamlTag: propName,
			Message: fmt.Sprintf("schema %s has %s property %q with no matching Go field",
				schemaDesc(loc), qualifier, propName),
		})
	}

	// Direction 2: Go field handling.
	for _, yamlName := range goFields.order {
		f := goFields.m[yamlName]
		_, omitempty, _, _ := parseYAMLTag(f.Tag.Get("yaml"))
		symbol := owner + "." + f.Name

		prop, ok := props[yamlName]
		if !ok {
			if closed {
				c.add(&finding{
					Severity: severityCritical, Entity: owner, Category: catFieldPresence,
					File: c.typesFile, Symbol: symbol, Field: f.Name, YamlTag: yamlName,
					Message: fmt.Sprintf("Go field %s (yaml %q) has no schema property and %s sets additionalProperties:false",
						symbol, yamlName, schemaDesc(loc)),
				})
			}

			continue
		}

		c.checkRequiredness(owner, symbol, f.Name, yamlName, required[yamlName], omitempty)

		if recurse {
			c.compareType(owner, symbol, f.Type, prop, loc)
		}
	}
}

// checkRequiredness verifies the omitempty tag agrees with schema requiredness.
func (c *checker) checkRequiredness(entity, symbol, field, yamlName string, required, omitempty bool) {
	switch {
	case required && omitempty:
		c.add(&finding{
			Severity: severityCritical, Entity: entity, Category: catRequiredOpt,
			File: c.typesFile, Symbol: symbol, Field: field, YamlTag: yamlName,
			Message: fmt.Sprintf("Go field %s is schema-required but has omitempty (remove omitempty)", symbol),
		})
	case !required && !omitempty:
		c.add(&finding{
			Severity: severityMajor, Entity: entity, Category: catRequiredOpt,
			File: c.typesFile, Symbol: symbol, Field: field, YamlTag: yamlName,
			Message: fmt.Sprintf("Go field %s is schema-optional but missing omitempty (add omitempty)", symbol),
		})
	}
}

// compareType checks that a Go field's type matches the schema property's
// shape, recursing into nested objects/arrays/maps.
func (c *checker) compareType(entity, symbol string, goType reflect.Type, prop *schemaNode, loc ref) {
	prop, loc = c.derefNode(symbol, prop, loc)
	if prop == nil {
		return
	}

	// NoteList is []string with a custom marshaler matching oneOf[string,array].
	if isNoteListType(goType) {
		if !isNoteListOneOf(prop) {
			c.typeMismatch(entity, symbol, "NoteList", prop, loc)
		}

		return
	}

	switch goType.Kind() {
	case reflect.Pointer:
		c.compareType(entity, symbol, goType.Elem(), prop, loc)
	case reflect.Slice:
		c.compareSlice(entity, symbol, goType, prop, loc)
	case reflect.Map:
		// All GLX maps are keyed by entity/vocab ID and serialize as objects.
		if c.expectType(entity, symbol, goType, prop, loc, schemaTypeObject) {
			c.compareMapValue(entity, symbol, goType.Elem(), prop, loc)
		}
	case reflect.Struct:
		c.compareStructOrOneOf(entity, symbol, goType, prop, loc)
	case reflect.Interface:
		// `any` (Properties values, TemporalValue.Value) accepts any schema.
	default:
		c.compareScalar(entity, symbol, goType, prop, loc)
	}
}

// compareScalar handles primitive Go kinds against their schema type.
func (c *checker) compareScalar(entity, symbol string, goType reflect.Type, prop *schemaNode, loc ref) {
	switch goType.Kind() {
	case reflect.String:
		c.expectType(entity, symbol, goType, prop, loc, schemaTypeString)
	case reflect.Bool:
		c.expectType(entity, symbol, goType, prop, loc, schemaTypeBoolean)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		c.expectType(entity, symbol, goType, prop, loc, schemaTypeInteger, schemaTypeNumber)
	case reflect.Float32, reflect.Float64:
		c.expectType(entity, symbol, goType, prop, loc, schemaTypeNumber)
	default:
		c.add(&finding{
			Severity: severityInfo, Entity: entity, Category: catFieldTypes,
			File: c.typesFile, Symbol: symbol,
			Message: fmt.Sprintf("Go field %s has unhandled kind %s", symbol, goType.Kind()),
		})
	}
}

// compareSlice checks an array field and recurses into its element type —
// whether that element is a struct ([]Participant), a scalar ([]string vs
// items.type), or any other shape compareType understands. Validating scalar
// items catches drift like []string vs schema items {type: integer}.
func (c *checker) compareSlice(entity, symbol string, goType reflect.Type, prop *schemaNode, loc ref) {
	if !c.expectType(entity, symbol, goType, prop, loc, schemaTypeArray) || prop.Items == nil {
		return
	}
	items, itemsLoc := c.derefNode(symbol, prop.Items, loc)
	if items != nil {
		c.compareType(entity, symbol, goType.Elem(), items, itemsLoc)
	}
}

// compareStructOrOneOf maps a Go struct to either an object schema or a oneOf
// of objects (the EntityRef "typed reference" pattern).
func (c *checker) compareStructOrOneOf(entity, symbol string, goType reflect.Type, prop *schemaNode, loc ref) {
	if len(prop.OneOf) > 0 {
		c.compareStruct(entity, goType, mergeOneOfObjects(prop.OneOf), loc, true)

		return
	}
	if c.expectType(entity, symbol, goType, prop, loc, schemaTypeObject) && len(prop.Properties) > 0 {
		c.compareStruct(entity, goType, prop, loc, true)
	}
}

// compareMapValue checks the value type of a map[string]T field against the
// schema's per-value subschema (patternProperties or additionalProperties).
// It handles struct values (map[string]*VocabularyEntry), scalar values
// (map[string]string vs a typed additionalProperties), and anything else
// compareType understands. A free-form object — map[string]any whose schema
// is `additionalProperties: true` — has no per-value subschema and is a leaf.
func (c *checker) compareMapValue(entity, symbol string, valType reflect.Type, prop *schemaNode, loc ref) {
	valSchema := singleValueSchema(prop)
	if valSchema == nil {
		return // free-form object (e.g. map[string]any Properties): nothing to compare.
	}
	valSchema, valLoc := c.derefNode(symbol, valSchema, loc)
	if valSchema != nil {
		c.compareType(entity, symbol, valType, valSchema, valLoc)
	}
}

// expectType records a type-mismatch finding unless the schema's effective type
// is one of allowed (or the schema is a oneOf, handled elsewhere). Returns true
// when the type matched.
func (c *checker) expectType(entity, symbol string, goType reflect.Type, prop *schemaNode, loc ref, allowed ...string) bool {
	got := effectiveSchemaType(prop)
	if got == "" || got == schemaTypeOneOf {
		// Untyped or union schema: not a plain-type mismatch we can assert on.
		return true
	}
	if slices.Contains(allowed, got) {
		return true
	}
	c.typeMismatch(entity, symbol, goType.String(), prop, loc)

	return false
}

func (c *checker) typeMismatch(entity, symbol, goTypeName string, prop *schemaNode, loc ref) {
	c.add(&finding{
		Severity: severityCritical, Entity: entity, Category: catFieldTypes,
		File: c.typesFile, Symbol: symbol,
		Message: fmt.Sprintf("Go field %s has type %s but %s defines type %q",
			symbol, goTypeName, schemaDesc(loc), effectiveSchemaType(prop)),
	})
}

// derefNode follows a single $ref and reports broken references as findings.
func (c *checker) derefNode(symbol string, n *schemaNode, loc ref) (*schemaNode, ref) {
	out, newLoc, err := c.schemas.deref(n, loc)
	if err != nil {
		c.add(&finding{
			Severity: severityCritical, Entity: "schema", Category: catInternal,
			File: c.typesFile, Symbol: symbol,
			Message: fmt.Sprintf("unresolved schema reference for %s: %v", symbol, err),
		})

		return nil, loc
	}

	return out, newLoc
}

// ---- helpers -------------------------------------------------------------

// yamlFields indexes a struct's exported fields by their yaml key, preserving
// declaration order for stable iteration.
type yamlFields struct {
	m     map[string]reflect.StructField
	order []string
}

func structFieldsByYAML(goType reflect.Type) yamlFields {
	out := yamlFields{m: make(map[string]reflect.StructField)}
	for f := range goType.Fields() {
		if !f.IsExported() {
			continue
		}
		name, _, inline, skip := parseYAMLTag(f.Tag.Get("yaml"))
		if skip || inline {
			continue
		}
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		out.m[name] = f
		out.order = append(out.order, name)
	}

	return out
}

// parseYAMLTag splits a `yaml:"name,opt1,opt2"` tag.
func parseYAMLTag(tag string) (name string, omitempty, inline, skip bool) {
	if tag == "" {
		return "", false, false, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "-" {
		skip = true
	}
	for _, p := range parts[1:] {
		switch p {
		case "omitempty":
			omitempty = true
		case "inline":
			inline = true
		}
	}

	return name, omitempty, inline, skip
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

func isNoteListType(t reflect.Type) bool {
	return derefType(t).Name() == "NoteList"
}

// isNoteListOneOf reports whether a schema node is the GLX note shape:
// oneOf[ {string}, {array of string} ].
func isNoteListOneOf(n *schemaNode) bool {
	if len(n.OneOf) != 2 {
		return false
	}
	var hasString, hasStringArray bool
	for _, b := range n.OneOf {
		switch b.Type {
		case schemaTypeString:
			hasString = true
		case schemaTypeArray:
			if b.Items != nil && b.Items.Type == schemaTypeString {
				hasStringArray = true
			}
		}
	}

	return hasString && hasStringArray
}

// effectiveSchemaType returns the node's declared type, inferring "object",
// "array", or "oneof" when type is omitted but structure implies it.
func effectiveSchemaType(n *schemaNode) string {
	if n.Type != "" {
		return n.Type
	}
	switch {
	case len(n.OneOf) > 0 || len(n.AnyOf) > 0:
		return schemaTypeOneOf
	case len(n.Properties) > 0 || len(n.PatternProperties) > 0 || n.additionalPropsSchema() != nil:
		return schemaTypeObject
	case n.Items != nil:
		return schemaTypeArray
	default:
		return ""
	}
}

// singleValueSchema returns the per-value subschema of an object-as-map node,
// drawn from patternProperties (the GLX entity/vocab maps) or
// additionalProperties (PropertyDefinition.fields).
func singleValueSchema(n *schemaNode) *schemaNode {
	for _, k := range sortedKeys(n.PatternProperties) {
		return n.PatternProperties[k]
	}

	return n.additionalPropsSchema()
}

// mergeOneOfObjects unions the properties of every object branch of a oneOf
// into a single closed object with no required fields — the correct shape to
// compare a Go "typed reference" struct (EntityRef) against, where exactly one
// field is set per instance but the struct carries all of them as optional.
func mergeOneOfObjects(branches []*schemaNode) *schemaNode {
	merged := &schemaNode{
		Type:                 schemaTypeObject,
		Properties:           map[string]*schemaNode{},
		AdditionalProperties: []byte(jsonFalse),
	}
	for _, b := range branches {
		maps.Copy(merged.Properties, b.Properties)
	}

	return merged
}

func stringSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, s := range items {
		out[s] = true
	}

	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}

// schemaDesc renders a schema location for human-readable messages.
func schemaDesc(loc ref) string {
	if loc.pointer == "" {
		return loc.file
	}

	return loc.file + "#" + loc.pointer
}
