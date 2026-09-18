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
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// chartTestArchive builds an archive from a parent -> child map, naming every
// person after its ID so chart assertions can match on the name.
func chartTestArchive(childrenOf map[string][]string) *glxlib.GLXFile {
	archive := &glxlib.GLXFile{
		Persons:       map[string]*glxlib.Person{},
		Relationships: map[string]*glxlib.Relationship{},
	}
	person := func(id string) {
		if _, ok := archive.Persons[id]; !ok {
			archive.Persons[id] = &glxlib.Person{Properties: map[string]any{"name": id}}
		}
	}
	for parent, children := range childrenOf {
		person(parent)
		for _, child := range children {
			person(child)
			archive.Relationships["rel-"+parent+"-"+child] = &glxlib.Relationship{
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: parent, Role: glxlib.ParticipantRoleParent},
					{Person: child, Role: glxlib.ParticipantRoleChild},
				},
			}
		}
	}

	return archive
}

// chartFor builds one chart the way the model build does.
func chartFor(archive *glxlib.GLXFile, personID, direction string) *personChart {
	return buildPersonChart(personID, direction, archive, newSiteIndex(archive))
}

// nodeNamed returns the first node with the given name, or nil.
func nodeNamed(chart *personChart, name string) *chartNode {
	for i := range chart.Nodes {
		if chart.Nodes[i].Name == name {
			return &chart.Nodes[i]
		}
	}

	return nil
}

// mustNodeNamed fails the test when the chart has no node with that name.
func mustNodeNamed(t *testing.T, chart *personChart, name string) *chartNode {
	t.Helper()
	node := nodeNamed(chart, name)
	if node == nil {
		t.Fatalf("chart has no node named %q", name)
	}

	return node
}

// columnOf converts a node's x coordinate back to its generation column.
func columnOf(node *chartNode) int {
	return (node.X - chartPad) / (chartNodeW + chartColGap)
}

func TestBuildPersonChart_AncestorsWalkParents(t *testing.T) {
	archive := chartTestArchive(map[string][]string{
		"grandfather": {"father"},
		"grandmother": {"father"},
		"father":      {"child"},
		"mother":      {"child"},
	})

	chart := chartFor(archive, "child", chartAncestors)
	if chart == nil {
		t.Fatal("expected a pedigree chart for a person with parents")
	}
	if got, want := len(chart.Nodes), 5; got != want {
		t.Errorf("node count = %d, want %d (self, 2 parents, 2 grandparents)", got, want)
	}
	if got, want := len(chart.Edges), 4; got != want {
		t.Errorf("edge count = %d, want %d", got, want)
	}

	for name, wantColumn := range map[string]int{"child": 0, "father": 1, "mother": 1, "grandfather": 2} {
		node := nodeNamed(chart, name)
		if node == nil {
			t.Errorf("chart missing %q", name)

			continue
		}
		if got := columnOf(node); got != wantColumn {
			t.Errorf("%q in column %d, want %d", name, got, wantColumn)
		}
	}

	root := mustNodeNamed(t, chart, "child")
	if !root.Root {
		t.Error("the chart's subject should be marked as the root node")
	}
	if root.File != "" {
		t.Errorf("root node links to %q; it is the page being rendered and must not link", root.File)
	}
	if father := mustNodeNamed(t, chart, "father"); father.File == "" {
		t.Error("ancestor nodes must link to their own profile page")
	}
}

func TestBuildPersonChart_DescendantsWalkChildren(t *testing.T) {
	archive := chartTestArchive(map[string][]string{
		"parent":  {"child-a", "child-b"},
		"child-a": {"grandchild"},
	})

	chart := chartFor(archive, "parent", chartDescendants)
	if chart == nil {
		t.Fatal("expected a descendancy chart for a person with children")
	}
	if got, want := len(chart.Nodes), 4; got != want {
		t.Errorf("node count = %d, want %d", got, want)
	}
	if node := mustNodeNamed(t, chart, "grandchild"); columnOf(node) != 2 {
		t.Errorf("grandchild in column %d, want 2", columnOf(node))
	}
	if strings.Contains(chart.Label, "Pedigree") {
		t.Errorf("descendancy chart labeled %q", chart.Label)
	}
}

func TestBuildPersonChart_NilWithoutRelatives(t *testing.T) {
	archive := chartTestArchive(map[string][]string{"parent": {"child"}})

	if chart := chartFor(archive, "parent", chartAncestors); chart != nil {
		t.Error("a person with no recorded parents should get no pedigree chart")
	}
	if chart := chartFor(archive, "child", chartDescendants); chart != nil {
		t.Error("a person with no recorded children should get no descendancy chart")
	}
	if chart := chartFor(archive, "person-absent", chartAncestors); chart != nil {
		t.Error("an unknown person should get no chart")
	}
}

func TestBuildPersonChart_StopsAtGenerationLimit(t *testing.T) {
	// A straight line of ancestors deeper than the chart's generation limit.
	childrenOf := map[string][]string{}
	for gen := range chartAncestorGens + 2 {
		childrenOf[fmt.Sprintf("gen-%d", gen+1)] = []string{fmt.Sprintf("gen-%d", gen)}
	}

	chart := chartFor(chartTestArchive(childrenOf), "gen-0", chartAncestors)
	if chart == nil {
		t.Fatal("expected a pedigree chart")
	}
	if got, want := len(chart.Nodes), chartAncestorGens; got != want {
		t.Errorf("node count = %d, want %d (the root counts as the first generation)", got, want)
	}
	if chart.Truncated {
		t.Error("a chart stopped by the generation limit is complete, not truncated")
	}
	for i := range chart.Nodes {
		node := &chart.Nodes[i]
		if col := columnOf(node); col >= chartAncestorGens {
			t.Errorf("node %q drawn in column %d, beyond the %d-generation limit", node.Name, col, chartAncestorGens)
		}
	}
}

func TestBuildPersonChart_TruncatesAtNodeBudget(t *testing.T) {
	// One parent with far more children than the node budget allows.
	children := make([]string, chartMaxNodes+20)
	for i := range children {
		children[i] = fmt.Sprintf("child-%03d", i)
	}

	chart := chartFor(chartTestArchive(map[string][]string{"parent": children}), "parent", chartDescendants)
	if chart == nil {
		t.Fatal("expected a descendancy chart")
	}
	if !chart.Truncated {
		t.Error("a chart cut short by the node budget must report itself as truncated")
	}
	if len(chart.Nodes) > chartMaxNodes {
		t.Errorf("node count = %d, exceeds the budget of %d", len(chart.Nodes), chartMaxNodes)
	}
}

func TestBuildPersonChart_SurvivesCyclicAncestry(t *testing.T) {
	// A malformed archive where a person is their own grandparent.
	archive := chartTestArchive(map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	})

	chart := chartFor(archive, "a", chartAncestors)
	if chart == nil {
		t.Fatal("expected a pedigree chart")
	}
	if len(chart.Nodes) > chartAncestorGens {
		t.Errorf("cycle produced %d nodes, more than the %d-generation limit allows", len(chart.Nodes), chartAncestorGens)
	}
}

func TestLayoutChart_CentersParentOnItsChildren(t *testing.T) {
	archive := chartTestArchive(map[string][]string{
		"father": {"child"},
		"mother": {"child"},
	})

	chart := chartFor(archive, "child", chartAncestors)
	root := mustNodeNamed(t, chart, "child")
	father := mustNodeNamed(t, chart, "father")
	mother := mustNodeNamed(t, chart, "mother")

	if want := (father.Y + mother.Y) / 2; root.Y != want {
		t.Errorf("root y = %d, want %d (centered between its parents)", root.Y, want)
	}
	if chart.Height <= mother.Y+chartNodeH {
		t.Errorf("chart height %d clips the lowest node (bottom edge %d)", chart.Height, mother.Y+chartNodeH)
	}
	if chart.Width <= father.X+chartNodeW {
		t.Errorf("chart width %d clips the rightmost node (right edge %d)", chart.Width, father.X+chartNodeW)
	}
}

func TestBuildPersonChart_TruncatesLongNames(t *testing.T) {
	archive := chartTestArchive(map[string][]string{"parent": {"child"}})
	long := strings.Repeat("Bartholomew", 5)
	archive.Persons["parent"].Properties["name"] = long

	chart := chartFor(archive, "child", chartAncestors)
	node := mustNodeNamed(t, chart, truncateRunes(long, chartNameMaxRunes))
	if !strings.HasSuffix(node.Name, "…") {
		t.Errorf("truncated name %q should end with an ellipsis", node.Name)
	}
}

func TestRenderSite_DrawsFamilyCharts(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), siteModelOptions{Title: "Smith Family"})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	mary := readFile(t, filepath.Join(out, "persons", "person-mary-smith.html"))
	for _, want := range []string{
		`<figure class="chart chart-ancestors">`,
		`aria-label="Pedigree chart for Mary Smith"`,
		`<a class="chart-node" href="person-john-smith.html">`,
		`<g class="chart-node is-root">`,
	} {
		if !strings.Contains(mary, want) {
			t.Errorf("Mary's profile missing %q", want)
		}
	}

	john := readFile(t, filepath.Join(out, "persons", "person-john-smith.html"))
	if !strings.Contains(john, `<figure class="chart chart-descendants">`) {
		t.Error("John's profile should include a descendancy chart")
	}
	if strings.Contains(john, `chart chart-ancestors`) {
		t.Error("John has no recorded parents and should have no pedigree chart")
	}

	living := readFile(t, filepath.Join(out, "persons", "person-living-soul.html"))
	if strings.Contains(living, "<figure class=\"chart") {
		t.Error("a person with no relatives should have no charts at all")
	}
}

func TestRenderSite_EscapesChartNames(t *testing.T) {
	archive := chartTestArchive(map[string][]string{"parent": {"child"}})
	archive.Persons["parent"].Properties["name"] = `<script>alert(1)</script>`

	model := buildSiteModel(archive, siteModelOptions{})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	page := readFile(t, filepath.Join(out, "persons", "child.html"))
	if strings.Contains(page, "<script>alert(1)</script>") {
		t.Error("a hostile name reached the chart SVG unescaped")
	}
}
