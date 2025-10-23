package test_suite

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func loadYAML(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	var out map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", path, err)
	}
	return out
}

func assertStringKey(t *testing.T, data map[string]interface{}, key string) {
	t.Helper()
	value, ok := data[key]
	if !ok {
		t.Fatalf("expected key %q to exist", key)
	}
	if _, ok := value.(string); !ok {
		t.Fatalf("expected key %q to be a string", key)
	}
}

func TestBasicFamilyPersons(t *testing.T) {
	dir := filepath.Join("..", "examples", "basic-family", "persons")
	persons := []string{
		"person-mother.glx",
		"person-father.glx",
		"person-child-alice.glx",
		"person-child-bob.glx",
	}

	for _, name := range persons {
		path := filepath.Join(dir, name)
		data := loadYAML(t, path)
		assertStringKey(t, data, "id")
		assertStringKey(t, data, "version")

		ci, ok := data["concluded_identity"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: concluded_identity should be an object", name)
		}
		assertStringKey(t, ci, "primary_name")
	}
}

func TestBasicFamilyRelationships(t *testing.T) {
	dir := filepath.Join("..", "examples", "basic-family", "relationships")
	relations := []string{
		"rel-marriage.glx",
		"rel-parent-alice.glx",
		"rel-parent-bob.glx",
	}

	for _, name := range relations {
		path := filepath.Join(dir, name)
		data := loadYAML(t, path)
		assertStringKey(t, data, "id")
		assertStringKey(t, data, "version")
		assertStringKey(t, data, "type")

		persons, ok := data["persons"].([]interface{})
		if !ok || len(persons) < 2 {
			t.Fatalf("%s: persons should be an array with at least two entries", name)
		}
	}
}

func TestBasicFamilyConfig(t *testing.T) {
	dir := filepath.Join("..", "examples", "basic-family", ".glx-archive")
	config := loadYAML(t, filepath.Join(dir, "config.glx"))
	assertStringKey(t, config, "version")
	assertStringKey(t, config, "schema")

	schema := loadYAML(t, filepath.Join(dir, "schema-version.glx"))
	assertStringKey(t, schema, "schema")
}

func TestMinimalExamplePerson(t *testing.T) {
	path := filepath.Join("..", "examples", "minimal", "persons", "person-abc123.glx")
	data := loadYAML(t, path)
	assertStringKey(t, data, "id")
	assertStringKey(t, data, "version")
}
