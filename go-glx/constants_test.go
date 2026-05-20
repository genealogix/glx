package glx

import "testing"

func TestRelationshipTypePossiblySamePersonConstant(t *testing.T) {
	if RelationshipTypePossiblySamePerson != "possibly_same_person" {
		t.Fatalf("RelationshipTypePossiblySamePerson = %q, want %q", RelationshipTypePossiblySamePerson, "possibly_same_person")
	}
}
