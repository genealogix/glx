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
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ChangeKind classifies how an entity changed between two archive states.
type ChangeKind string

const (
	ChangeAdded    ChangeKind = "added"
	ChangeModified ChangeKind = "modified"
	ChangeRemoved  ChangeKind = "removed"
)

// FieldChange describes a single field-level difference within an entity.
type FieldChange struct {
	Path     string `json:"path"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// EntityChange describes a single entity-level difference.
type EntityChange struct {
	Kind       ChangeKind    `json:"kind"`
	EntityType string        `json:"entity_type"`
	ID         string        `json:"id"`
	Summary    string        `json:"summary"`
	Fields     []FieldChange `json:"fields,omitempty"`
}

// DiffStats summarizes aggregate metrics about the diff.
type DiffStats struct {
	Added                int `json:"added"`
	Modified             int `json:"modified"`
	Removed              int `json:"removed"`
	ConfidenceUpgrades   int `json:"confidence_upgrades"`
	ConfidenceDowngrades int `json:"confidence_downgrades"`
	NewSources           int `json:"new_sources"`
	NewCitations         int `json:"new_citations"`
}

// DiffResult holds the complete comparison between two archive states.
type DiffResult struct {
	Changes []EntityChange `json:"changes"`
	Stats   DiffStats      `json:"stats"`
}

// confidenceRank maps confidence levels to numeric rank for upgrade/downgrade detection.
var confidenceRank = map[string]int{
	ConfidenceLevelLow:      0,
	ConfidenceLevelDisputed: 1,
	ConfidenceLevelMedium:   2,
	ConfidenceLevelHigh:     3,
}

// DiffArchives compares two loaded archives and returns a structured diff.
// If personFilter is non-empty, only changes relevant to that person are included.
func DiffArchives(oldArchive, newArchive *GLXFile, personFilter string) *DiffResult {
	result := &DiffResult{}

	diffEntityMap(result, EntityTypePersons, oldArchive.Persons, newArchive.Persons)
	diffEntityMap(result, EntityTypeEvents, oldArchive.Events, newArchive.Events)
	diffEntityMap(result, EntityTypeRelationships, oldArchive.Relationships, newArchive.Relationships)
	diffEntityMap(result, EntityTypePlaces, oldArchive.Places, newArchive.Places)
	diffEntityMap(result, EntityTypeSources, oldArchive.Sources, newArchive.Sources)
	diffEntityMap(result, EntityTypeCitations, oldArchive.Citations, newArchive.Citations)
	diffEntityMap(result, EntityTypeRepositories, oldArchive.Repositories, newArchive.Repositories)
	diffEntityMap(result, EntityTypeAssertions, oldArchive.Assertions, newArchive.Assertions)
	diffEntityMap(result, EntityTypeMedia, oldArchive.Media, newArchive.Media)

	// Filter by person if requested (before computing stats)
	if personFilter != "" {
		filterByPerson(result, personFilter, oldArchive, newArchive)
	}

	// Compute stats on the (possibly filtered) change set
	computeStats(result)

	// Sort changes by entity type then ID
	sort.SliceStable(result.Changes, func(i, j int) bool {
		if result.Changes[i].EntityType != result.Changes[j].EntityType {
			return entityTypeOrder(result.Changes[i].EntityType) < entityTypeOrder(result.Changes[j].EntityType)
		}
		return result.Changes[i].ID < result.Changes[j].ID
	})

	return result
}

// entityTypeOrder returns a sort key for entity types to group them logically.
func entityTypeOrder(t string) int {
	order := map[string]int{
		EntityTypePersons:       0,
		EntityTypeEvents:        1,
		EntityTypeRelationships: 2,
		EntityTypePlaces:        3,
		EntityTypeAssertions:    4,
		EntityTypeSources:       5,
		EntityTypeCitations:     6,
		EntityTypeRepositories:  7,
		EntityTypeMedia:         8,
	}
	if v, ok := order[t]; ok {
		return v
	}
	return 99
}

// diffEntityMap compares two entity maps and appends changes to the result.
func diffEntityMap[T any](result *DiffResult, entityType string, oldMap, newMap map[string]*T) {
	// Check for added and modified entities
	for id, newEntity := range newMap {
		if oldEntity, exists := oldMap[id]; exists {
			// Possibly modified
			fields := compareEntity(oldEntity, newEntity)
			if len(fields) > 0 {
				result.Changes = append(result.Changes, EntityChange{
					Kind:       ChangeModified,
					EntityType: entityType,
					ID:         id,
					Summary:    summarizeModified(entityType, id, fields),
					Fields:     fields,
				})
			}
		} else {
			// Added
			result.Changes = append(result.Changes, EntityChange{
				Kind:       ChangeAdded,
				EntityType: entityType,
				ID:         id,
				Summary:    summarizeEntity(entityType, id, newEntity),
			})
		}
	}

	// Check for removed entities
	for id := range oldMap {
		if _, exists := newMap[id]; !exists {
			result.Changes = append(result.Changes, EntityChange{
				Kind:       ChangeRemoved,
				EntityType: entityType,
				ID:         id,
				Summary:    summarizeEntity(entityType, id, oldMap[id]),
			})
		}
	}
}

// compareEntity compares two entities via YAML round-trip and returns field changes.
func compareEntity[T any](oldEntity, newEntity *T) []FieldChange {
	oldMap, oldErr := toYAMLMap(oldEntity)
	newMap, newErr := toYAMLMap(newEntity)

	if oldErr != nil || newErr != nil {
		return []FieldChange{
			{
				Path:     "",
				OldValue: formatSerializationStatus(oldErr),
				NewValue: formatSerializationStatus(newErr),
			},
		}
	}

	return diffMaps("", oldMap, newMap)
}

// formatSerializationStatus returns a human-readable description of the
// serialization status for use in FieldChange values.
func formatSerializationStatus(err error) string {
	if err == nil {
		return "(serializable)"
	}
	return fmt.Sprintf("(unserializable: %v)", err)
}

// toYAMLMap converts a struct to a map[string]any via YAML round-trip.
func toYAMLMap(v any) (map[string]any, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("yaml marshal: %w", err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	return m, nil
}

// diffMaps recursively compares two maps and returns field changes.
func diffMaps(prefix string, oldMap, newMap map[string]any) []FieldChange {
	var changes []FieldChange

	// Collect all keys from both maps
	allKeys := make(map[string]bool)
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	sortedAllKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedAllKeys = append(sortedAllKeys, k)
	}
	sort.Strings(sortedAllKeys)

	for _, key := range sortedAllKeys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		oldVal, oldExists := oldMap[key]
		newVal, newExists := newMap[key]

		if !oldExists {
			changes = append(changes, FieldChange{
				Path:     path,
				OldValue: "(none)",
				NewValue: formatValue(newVal),
			})
			continue
		}
		if !newExists {
			changes = append(changes, FieldChange{
				Path:     path,
				OldValue: formatValue(oldVal),
				NewValue: "(none)",
			})
			continue
		}

		// Both exist — compare recursively if both are maps
		oldSubMap, oldIsMap := oldVal.(map[string]any)
		newSubMap, newIsMap := newVal.(map[string]any)
		if oldIsMap && newIsMap {
			changes = append(changes, diffMaps(path, oldSubMap, newSubMap)...)
			continue
		}

		// Compare as formatted strings
		if formatValue(oldVal) != formatValue(newVal) {
			changes = append(changes, FieldChange{
				Path:     path,
				OldValue: formatValue(oldVal),
				NewValue: formatValue(newVal),
			})
		}
	}

	return changes
}

// formatValue converts an arbitrary value to a display string.
func formatValue(v any) string {
	if v == nil {
		return "(none)"
	}

	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = formatValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		// Flatten to JSON-like string for display
		data, err := yaml.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return strings.TrimSpace(string(data))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// summarizeEntity generates a human-readable one-liner for an entity.
func summarizeEntity[T any](entityType, id string, entity *T) string {
	m, err := toYAMLMap(entity)
	if err != nil || m == nil {
		return id
	}

	switch entityType {
	case EntityTypePersons:
		return summarizePerson(id, m)
	case EntityTypeEvents:
		return summarizeEvent(m)
	case EntityTypeAssertions:
		return summarizeAssertion(m)
	case EntityTypeSources:
		return summarizeSource(m)
	case EntityTypeCitations:
		return summarizeCitation(m)
	case EntityTypeRelationships:
		return summarizeRelationship(m)
	case EntityTypePlaces:
		return summarizePlace(m)
	default:
		return id
	}
}

func summarizePerson(id string, m map[string]any) string {
	props, _ := m["properties"].(map[string]any)
	if props == nil {
		return id
	}

	name, _ := props["name"].(string)
	if name == "" {
		return id
	}

	var parts []string
	if born, ok := props["born_on"].(string); ok && born != "" {
		parts = append(parts, "b. "+born)
	}
	if died, ok := props["died_on"].(string); ok && died != "" {
		parts = append(parts, "d. "+died)
	}
	if len(parts) > 0 {
		return name + " (" + strings.Join(parts, "; ") + ")"
	}
	return name
}

func summarizeEvent(m map[string]any) string {
	eventType, _ := m["type"].(string)
	date, _ := m["date"].(string)
	title, _ := m["title"].(string)

	if title != "" {
		return title
	}

	parts := make([]string, 0, 2)
	if eventType != "" {
		parts = append(parts, eventType)
	}
	if date != "" {
		parts = append(parts, date)
	}
	return strings.Join(parts, ", ")
}

func summarizeAssertion(m map[string]any) string {
	prop, _ := m["property"].(string)
	val, _ := m["value"].(string)
	conf, _ := m["confidence"].(string)

	var sb strings.Builder
	if prop != "" {
		sb.WriteString(prop)
		if val != "" {
			sb.WriteString(" = ")
			sb.WriteString(val)
		}
	}
	if conf != "" {
		if sb.Len() > 0 {
			sb.WriteString(" (")
			sb.WriteString(conf)
			sb.WriteString(" confidence)")
		} else {
			sb.WriteString(conf)
			sb.WriteString(" confidence")
		}
	}
	return sb.String()
}

func summarizeSource(m map[string]any) string {
	title, _ := m["title"].(string)
	if title != "" {
		return fmt.Sprintf("%q", title)
	}
	return "(untitled source)"
}

func summarizeCitation(m map[string]any) string {
	source, _ := m["source"].(string)
	props, _ := m["properties"].(map[string]any)
	var locator string
	if props != nil {
		locator, _ = props["locator"].(string)
	}
	if source != "" && locator != "" {
		return source + " @ " + locator
	}
	if source != "" {
		return source
	}
	return "(citation)"
}

func summarizeRelationship(m map[string]any) string {
	relType, _ := m["type"].(string)
	if relType == "" {
		return "(relationship)"
	}
	return relType
}

func summarizePlace(m map[string]any) string {
	name, _ := m["name"].(string)
	if name != "" {
		return name
	}
	return "(unnamed place)"
}

// summarizeModified generates a one-line summary for a modified entity.
func summarizeModified(entityType, id string, fields []FieldChange) string {
	if len(fields) == 1 {
		f := fields[0]
		return fmt.Sprintf("%s %s: %s: %s → %s", entityType, id, f.Path, f.OldValue, f.NewValue)
	}
	return fmt.Sprintf("%s %s: %d fields changed", entityType, id, len(fields))
}

// computeStats populates the DiffStats from the current change set.
func computeStats(result *DiffResult) {
	result.Stats = DiffStats{}
	for _, c := range result.Changes {
		switch c.Kind {
		case ChangeAdded:
			result.Stats.Added++
			if c.EntityType == EntityTypeSources {
				result.Stats.NewSources++
			}
			if c.EntityType == EntityTypeCitations {
				result.Stats.NewCitations++
			}
		case ChangeModified:
			result.Stats.Modified++
		case ChangeRemoved:
			result.Stats.Removed++
		}
	}
	computeConfidenceStats(result)
}

// computeConfidenceStats counts confidence upgrades and downgrades.
// Only counts changes where both old and new values are recognized confidence levels.
func computeConfidenceStats(result *DiffResult) {
	for _, c := range result.Changes {
		if c.EntityType != EntityTypeAssertions || c.Kind != ChangeModified {
			continue
		}
		for _, f := range c.Fields {
			if f.Path != "confidence" {
				continue
			}
			oldRank, oldOK := confidenceRank[strings.Trim(f.OldValue, "\"")]
			newRank, newOK := confidenceRank[strings.Trim(f.NewValue, "\"")]
			if !oldOK || !newOK {
				continue
			}
			if newRank > oldRank {
				result.Stats.ConfidenceUpgrades++
			} else if newRank < oldRank {
				result.Stats.ConfidenceDowngrades++
			}
		}
	}
}

// filterByPerson keeps only changes relevant to a specific person.
func filterByPerson(result *DiffResult, personID string, oldArchive, newArchive *GLXFile) {
	filtered := make([]EntityChange, 0, len(result.Changes))
	for _, c := range result.Changes {
		if entityReferencePerson(c, personID, oldArchive, newArchive) {
			filtered = append(filtered, c)
		}
	}
	result.Changes = filtered
}

// entityReferencePerson checks if a change is relevant to a specific person.
func entityReferencePerson(c EntityChange, personID string, oldArchive, newArchive *GLXFile) bool {
	if c.EntityType == EntityTypePersons {
		return c.ID == personID
	}

	if c.EntityType == EntityTypeEvents {
		if ev, ok := newArchive.Events[c.ID]; ok && ev != nil && eventHasParticipant(ev, personID) {
			return true
		}
		if ev, ok := oldArchive.Events[c.ID]; ok && ev != nil && eventHasParticipant(ev, personID) {
			return true
		}
	}

	if c.EntityType == EntityTypeRelationships {
		if rel, ok := newArchive.Relationships[c.ID]; ok && rel != nil && relationshipHasParticipant(rel, personID) {
			return true
		}
		if rel, ok := oldArchive.Relationships[c.ID]; ok && rel != nil && relationshipHasParticipant(rel, personID) {
			return true
		}
	}

	if c.EntityType == EntityTypeAssertions {
		if a, ok := newArchive.Assertions[c.ID]; ok && a != nil && a.Subject.Person == personID {
			return true
		}
		if a, ok := oldArchive.Assertions[c.ID]; ok && a != nil && a.Subject.Person == personID {
			return true
		}
	}

	return false
}

func eventHasParticipant(ev *Event, personID string) bool {
	if ev == nil {
		return false
	}
	for _, p := range ev.Participants {
		if p.Person == personID {
			return true
		}
	}
	return false
}

func relationshipHasParticipant(rel *Relationship, personID string) bool {
	if rel == nil {
		return false
	}
	for _, p := range rel.Participants {
		if p.Person == personID {
			return true
		}
	}
	return false
}
