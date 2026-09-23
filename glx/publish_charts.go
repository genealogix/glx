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

	glxlib "github.com/genealogix/glx/go-glx"
)

// Chart directions. The value is rendered into the SVG's accessible label and
// selects which adjacency list the layout walks.
const (
	chartAncestors   = "ancestors"
	chartDescendants = "descendants"
)

// Chart depth and size budget.
//
// A pedigree doubles every generation and a descendancy chart can fan out
// without bound, so both are capped: the generation limits keep the common
// case readable on a phone, and chartMaxNodes stops a densely-recorded family
// from emitting a megabyte of SVG onto a single profile page. The root counts
// as the first generation, so chartAncestorGens of 4 draws a person, their
// parents, grandparents, and great-grandparents.
const (
	chartAncestorGens   = 4
	chartDescendantGens = 3
	chartMaxNodes       = 60
)

// SVG layout geometry, in user units (which render as CSS pixels). Generations
// run left to right, one column each, with siblings stacked vertically.
const (
	chartNodeW    = 168
	chartNodeH    = 40
	chartColGap   = 40
	chartRowGap   = 12
	chartPad      = 8
	chartRowPitch = chartNodeH + chartRowGap
)

// Text baselines inside a node box, as offsets from its top-left corner. The
// layout computes absolute text coordinates from these so the template needs
// no arithmetic of its own.
const (
	chartTextInset = 10
	chartNameBase  = 17
	chartDatesBase = 32
)

// chartNameMaxRunes bounds a name to what fits inside a node box. Measuring
// text is impossible without a font, so the limit is by rune count, chosen for
// the box width at the stylesheet's chart font size.
const chartNameMaxRunes = 22

// personChart is a laid-out pedigree or descendancy chart, ready to render as
// inline SVG. Coordinates are absolute within the chart's own viewBox, so the
// template needs no layout logic of its own.
type personChart struct {
	Direction string // chartAncestors or chartDescendants
	Label     string // accessible label / <title> for the SVG
	Width     int
	Height    int
	Nodes     []chartNode
	Edges     []chartEdge
	// Truncated reports that the size budget stopped the walk before every
	// relative at the requested depth was drawn.
	Truncated bool
}

// chartNode is one person box in a chart. X/Y are the box's top-left corner
// and W/H its size; TextX, NameY, and DatesY are the absolute baselines of its
// two text lines. Every coordinate the template needs is precomputed here, so
// the markup carries no arithmetic.
type chartNode struct {
	ID     string
	Name   string
	Dates  string
	File   string // profile page filename; "" for the chart's root person
	X      int
	Y      int
	W      int
	H      int
	TextX  int
	NameY  int
	DatesY int
	Root   bool
}

// chartEdge is the connector drawn between a node and one of its relatives,
// as an SVG path built only from layout integers.
type chartEdge struct {
	Path string
}

// chartTree is the intermediate tree the layout pass walks. It is built from
// the relationship index before any coordinates are known.
type chartTree struct {
	id       string
	name     string
	dates    string
	file     string
	depth    int
	children []*chartTree
}

// buildPersonChart assembles a pedigree (ancestors) or descendancy
// (descendants) chart rooted at personID, or nil when the person has no
// relatives in that direction and there would be nothing to draw.
func buildPersonChart(personID, direction string, archive *glxlib.GLXFile, idx *siteIndex) *personChart {
	maxGen := chartAncestorGens
	if direction == chartDescendants {
		maxGen = chartDescendantGens
	}

	budget := chartMaxNodes
	root, truncated := buildChartTree(personID, direction, 0, maxGen, archive, idx, map[string]bool{}, &budget)
	if root == nil || len(root.children) == 0 {
		return nil
	}
	root.file = "" // the root is the page the chart is drawn on

	chart := &personChart{
		Direction: direction,
		Label:     chartLabel(direction, root.name),
		Truncated: truncated,
	}
	layoutChart(chart, root)

	return chart
}

// chartLabel builds the SVG's accessible label.
func chartLabel(direction, name string) string {
	if direction == chartDescendants {
		return "Descendancy chart for " + name
	}

	return "Pedigree chart for " + name
}

// buildChartTree walks parents (ancestors) or children (descendants) breadth
// by depth, stopping at maxGen generations or when the node budget runs out.
// `path` holds the ancestry of the current node so a person may legitimately
// appear in two branches (pedigree collapse) while a malformed self-ancestor
// cycle still terminates.
func buildChartTree(
	personID, direction string,
	depth, maxGen int,
	archive *glxlib.GLXFile,
	idx *siteIndex,
	path map[string]bool,
	budget *int,
) (node *chartTree, truncated bool) {
	person, ok := archive.Persons[personID]
	if !ok || person == nil {
		return nil, false
	}
	if *budget <= 0 {
		return nil, true
	}
	*budget--

	birth, death := vitalEvents(personID, archive)
	node = &chartTree{
		id:    personID,
		name:  truncateRunes(extractPersonName(person), chartNameMaxRunes),
		dates: lifeSpan(birth, death),
		file:  idx.files[personID],
		depth: depth,
	}

	// depth is 0-based and the root is the first generation, so a chart of
	// maxGen generations fills depths 0..maxGen-1.
	if depth >= maxGen-1 || path[personID] {
		return node, false
	}

	path[personID] = true
	defer delete(path, personID)

	nextIDs := idx.parents[personID]
	if direction == chartDescendants {
		nextIDs = idx.children[personID]
	}
	for _, nextID := range nextIDs {
		child, childTruncated := buildChartTree(nextID, direction, depth+1, maxGen, archive, idx, path, budget)
		truncated = truncated || childTruncated
		if child != nil {
			node.children = append(node.children, child)
		}
	}

	return node, truncated
}

// layoutChart assigns coordinates to every node and builds the connectors.
// Columns are fixed by depth; rows are packed so that leaves sit on an even
// pitch and every parent is centered on the block of relatives it connects to.
func layoutChart(chart *personChart, root *chartTree) {
	nextY := chartPad
	maxDepth := 0
	positions := map[*chartTree]int{}

	var assign func(t *chartTree) int
	assign = func(t *chartTree) int {
		if t.depth > maxDepth {
			maxDepth = t.depth
		}
		if len(t.children) == 0 {
			y := nextY
			nextY += chartRowPitch
			positions[t] = y

			return y
		}

		first := assign(t.children[0])
		last := first
		for _, child := range t.children[1:] {
			last = assign(child)
		}
		y := (first + last) / 2
		positions[t] = y

		return y
	}
	assign(root)

	var emit func(t *chartTree)
	emit = func(t *chartTree) {
		x := chartPad + t.depth*(chartNodeW+chartColGap)
		y := positions[t]
		chart.Nodes = append(chart.Nodes, chartNode{
			ID:     t.id,
			Name:   t.name,
			Dates:  t.dates,
			File:   t.file,
			X:      x,
			Y:      y,
			W:      chartNodeW,
			H:      chartNodeH,
			TextX:  x + chartTextInset,
			NameY:  y + chartNameBase,
			DatesY: y + chartDatesBase,
			Root:   t.depth == 0,
		})
		for _, child := range t.children {
			chart.Edges = append(chart.Edges, chartEdge{
				Path: elbowPath(x+chartNodeW, y+chartNodeH/2,
					chartPad+child.depth*(chartNodeW+chartColGap), positions[child]+chartNodeH/2),
			})
			emit(child)
		}
	}
	emit(root)

	chart.Width = chartPad*2 + (maxDepth+1)*chartNodeW + maxDepth*chartColGap
	chart.Height = nextY - chartRowGap + chartPad
}

// elbowPath draws the connector between two node edges: out horizontally to
// the midpoint of the column gap, vertically to the target row, then in to the
// target node.
func elbowPath(x1, y1, x2, y2 int) string {
	mid := (x1 + x2) / 2

	return fmt.Sprintf("M%d %d H%d V%d H%d", x1, y1, mid, y2, x2)
}
