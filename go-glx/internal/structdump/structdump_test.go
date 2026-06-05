package structdump

import "testing"

const typesGo = "../../types.go"

func TestExtractCollections(t *testing.T) {
	d, err := Extract(typesGo)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// Known GLXFile collections (yaml key -> Go type). These are stable anchors.
	want := map[string]string{
		"persons":           "Person",
		"events":            "Event",
		"event_types":       "VocabularyEntry",
		"person_properties": "PropertyDefinition",
	}
	got := map[string]string{}
	for _, c := range d.Collections {
		got[c.YAMLKey] = c.GoType
		if c.Line <= 0 {
			t.Errorf("collection %q has non-positive line %d", c.YAMLKey, c.Line)
		}
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("collection %q: got Go type %q, want %q", k, got[k], v)
		}
	}

	// The *Metadata field and the unexported validation field are NOT collections.
	for _, c := range d.Collections {
		if c.YAMLKey == "metadata" || c.GoType == "Metadata" {
			t.Errorf("non-collection field leaked in as a collection: %+v", c)
		}
	}
}

func TestReferentialIntegrity(t *testing.T) {
	d, err := Extract(typesGo)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(d.Collections) == 0 {
		t.Fatal("no collections extracted")
	}
	for _, c := range d.Collections {
		if _, ok := d.Types[c.GoType]; !ok {
			t.Errorf("collection %q points at Go type %q which was not extracted", c.YAMLKey, c.GoType)
		}
	}
}

func TestFields(t *testing.T) {
	d, err := Extract(typesGo)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	person, ok := d.CollectionType("persons")
	if !ok {
		t.Fatal(`CollectionType("persons") not found`)
	}
	tags := map[string]int{}
	for _, f := range person.Fields {
		tags[f.YAMLTag] = f.Line
	}
	for _, tag := range []string{"properties", "notes"} {
		if tags[tag] == 0 {
			t.Errorf("Person missing field with yaml tag %q (have %v)", tag, tags)
		}
	}
}
