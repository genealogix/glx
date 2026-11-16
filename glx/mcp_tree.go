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
	"context"
	"errors"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

const (
	mcpTreeDefaultGenerations = 4
	mcpTreeMaxGenerations     = 20
	mcpTreeMaxNodes           = 1000
)

var errMCPTreeGenerations = errors.New("generations must be between 0 and 20; zero uses the default of 4")

type mcpTreeResult struct {
	*treeJSONResult
	Generations int  `json:"generations"`
	NodeCount   int  `json:"node_count"`
	Truncated   bool `json:"truncated"`
}

type mcpTreeWalker struct {
	tree      *treeContext
	ancestors bool
	maxDepth  int
	nodes     int
	truncated bool
}

func (b *mcpArchiveBackend) tree(ctx context.Context, a glxmcp.TreeArgs, ancestors bool) (*mcpTreeResult, error) {
	if a.Generations < 0 || a.Generations > mcpTreeMaxGenerations {
		return nil, errMCPTreeGenerations
	}
	if a.Generations == 0 {
		a.Generations = mcpTreeDefaultGenerations
	}
	id, _, err := resolveJSONPerson(b.archive, a.Person)
	if err != nil {
		return nil, err
	}
	tree, err := newMCPTreeContext(ctx, b.archive)
	if err != nil {
		return nil, err
	}
	walker := mcpTreeWalker{tree: tree, ancestors: ancestors, maxDepth: a.Generations}
	root, err := walker.build(ctx, id, 0, map[string]bool{})
	if err != nil {
		return nil, err
	}
	direction := jsonDescendants
	if ancestors {
		direction = jsonAncestors
	}

	return &mcpTreeResult{treeJSONResult: &treeJSONResult{Direction: direction, Root: root}, Generations: a.Generations, NodeCount: walker.nodes, Truncated: walker.truncated}, nil
}

func (w *mcpTreeWalker) build(ctx context.Context, id string, depth int, visited map[string]bool) (*treeNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	node := makeTreeNode(w.tree.archive, id)
	node.Children = []*treeNode{}
	w.nodes++
	if visited[id] {
		return node, nil
	}
	visited[id] = true
	defer delete(visited, id)
	links := findChildren(w.tree, id)
	if w.ancestors {
		links = findParents(w.tree, id)
	}
	if depth >= w.maxDepth {
		w.truncated = w.truncated || len(links) > 0

		return node, nil
	}
	for _, link := range links {
		if w.nodes >= mcpTreeMaxNodes {
			w.truncated = true

			break
		}
		child, err := w.build(ctx, link.personID, depth+1, visited)
		if err != nil {
			return nil, err
		}
		child.RelType, child.Step, child.Hypothetical = link.relType, link.step, link.hypothetical
		node.Children = append(node.Children, child)
	}

	return node, nil
}

func newMCPTreeContext(ctx context.Context, archive *glxlib.GLXFile) (*treeContext, error) {
	links, err := glxlib.ParentChildLinksContext(ctx, archive, nil)
	if err != nil {
		return nil, err
	}
	tree := &treeContext{archive: archive, parents: make(map[string][]treeRelPerson), children: make(map[string][]treeRelPerson)}
	for _, link := range links {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if link.Standing == glxlib.RelationshipStandingDisproven {
			continue
		}
		tree.parents[link.ChildID] = append(tree.parents[link.ChildID], treeRelPersonFromLink(&link, link.ParentID))
		tree.children[link.ParentID] = append(tree.children[link.ParentID], treeRelPersonFromLink(&link, link.ChildID))
	}
	for _, index := range []map[string][]treeRelPerson{tree.parents, tree.children} {
		for id := range index {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			sortRelatedPersons(index[id])
		}
	}

	return tree, ctx.Err()
}
