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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// pathHop represents one step in the relationship path between two persons.
type pathHop struct {
	PersonID       string `json:"person_id"`
	PersonName     string `json:"person_name"`
	RelationshipID string `json:"relationship_id,omitempty"`
	RelType        string `json:"relationship_type,omitempty"`
	Role           string `json:"role,omitempty"`
}

// pathResult holds the full path output.
type pathResult struct {
	From    string    `json:"from"`
	To      string    `json:"to"`
	Hops    int       `json:"hops"`
	Path    []pathHop `json:"path"`
	Message string    `json:"message,omitempty"`
}

// pathEdge represents an adjacency in the relationship graph.
type pathEdge struct {
	PersonID       string
	RelationshipID string
	RelType        string
	Role           string // role of the source person in the relationship
}

// showPath loads an archive and finds the shortest path between two persons.
func showPath(archivePath, fromQuery, toQuery string, maxHops int, jsonOutput bool) error {
	if maxHops < 1 {
		return fmt.Errorf("--max-hops must be at least 1, got %d", maxHops)
	}

	archive, err := loadArchiveForPath(archivePath)
	if err != nil {
		return err
	}

	fromID, err := resolvePersonForPath(archive, fromQuery)
	if err != nil {
		return fmt.Errorf("from person: %w", err)
	}

	toID, err := resolvePersonForPath(archive, toQuery)
	if err != nil {
		return fmt.Errorf("to person: %w", err)
	}

	if fromID == toID {
		return fmt.Errorf("from and to resolve to the same person: %s", fromID)
	}

	adj := buildPathAdjacency(archive)
	path := bfsPath(fromID, toID, adj, maxHops)

	result := buildPathResult(fromID, toID, path, archive)

	if jsonOutput {
		return printPathJSON(result)
	}

	printPathText(result)
	return nil
}

// loadArchiveForPath loads an archive from a path.
func loadArchiveForPath(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, loadErr := LoadArchiveWithOptions(path, false)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}
		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// resolvePersonForPath finds a person by exact ID or name substring.
func resolvePersonForPath(archive *glxlib.GLXFile, query string) (string, error) {
	if person, ok := archive.Persons[query]; ok && person != nil {
		return query, nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []string

	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		name := extractPersonName(person)
		if strings.Contains(strings.ToLower(name), lowerQuery) {
			matches = append(matches, id)
		}
	}

	sort.Strings(matches)

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no person found matching %q", query)
	case 1:
		return matches[0], nil
	default:
		var lines []string
		for _, id := range matches {
			name := extractPersonName(archive.Persons[id])
			lines = append(lines, fmt.Sprintf("  %s  %s", id, name))
		}
		return "", fmt.Errorf("multiple persons match %q:\n%s\nUse exact person ID", query, strings.Join(lines, "\n"))
	}
}

// buildPathAdjacency builds an adjacency list from all relationships.
// Each person maps to a list of edges representing who they're connected to.
// Uses O(k²) pairwise edges per relationship, which is efficient for genealogy
// archives where relationships typically have 2-3 participants.
func buildPathAdjacency(archive *glxlib.GLXFile) map[string][]pathEdge {
	adj := make(map[string][]pathEdge)

	relIDs := sortedKeys(archive.Relationships)
	for _, relID := range relIDs {
		rel := archive.Relationships[relID]
		if rel == nil {
			continue
		}

		// Connect every participant to every other participant in this relationship
		for i, pi := range rel.Participants {
			if pi.Person == "" {
				continue
			}
			for j, pj := range rel.Participants {
				if i == j || pj.Person == "" {
					continue
				}
				adj[pi.Person] = append(adj[pi.Person], pathEdge{
					PersonID:       pj.Person,
					RelationshipID: relID,
					RelType:        rel.Type,
					Role:           pi.Role,
				})
			}
		}
	}

	return adj
}

// bfsNode tracks the BFS frontier with back-pointers to reconstruct the path.
type bfsNode struct {
	PersonID string
	Edge     *pathEdge // nil for the start node
	Parent   *bfsNode
	Depth    int
}

// bfsPath performs BFS from start to goal, returning the path as a slice of
// (personID, edge) pairs. Returns nil if no path found within maxHops.
func bfsPath(startID, goalID string, adj map[string][]pathEdge, maxHops int) []*bfsNode {
	if startID == goalID {
		return nil
	}

	visited := map[string]bool{startID: true}
	queue := []*bfsNode{{PersonID: startID}}
	head := 0

	for head < len(queue) {
		current := queue[head]
		queue[head] = nil // allow GC of dequeued nodes
		head++

		if current.Depth >= maxHops {
			continue
		}

		for _, edge := range adj[current.PersonID] {
			if visited[edge.PersonID] {
				continue
			}

			edgeCopy := edge
			next := &bfsNode{
				PersonID: edge.PersonID,
				Edge:     &edgeCopy,
				Parent:   current,
				Depth:    current.Depth + 1,
			}

			if edge.PersonID == goalID {
				return reconstructPath(next)
			}

			visited[edge.PersonID] = true
			queue = append(queue, next)
		}
	}

	return nil
}

// reconstructPath walks back-pointers to build the path from start to goal.
func reconstructPath(goal *bfsNode) []*bfsNode {
	var path []*bfsNode
	for n := goal; n != nil; n = n.Parent {
		path = append(path, n)
	}

	// Reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}

// buildPathResult converts BFS output into a display result.
func buildPathResult(fromID, toID string, path []*bfsNode, archive *glxlib.GLXFile) *pathResult {
	result := &pathResult{
		From: fromID,
		To:   toID,
	}

	if path == nil {
		result.Message = "No path found"
		result.Path = []pathHop{}
		return result
	}

	result.Hops = len(path) - 1

	// Build hops with relationship info on the source person, not the destination.
	for i, node := range path {
		hop := pathHop{
			PersonID:   node.PersonID,
			PersonName: pathPersonName(archive, node.PersonID),
		}
		// Attach the next node's edge info to this hop (the source of that edge).
		if i+1 < len(path) && path[i+1].Edge != nil {
			hop.RelationshipID = path[i+1].Edge.RelationshipID
			hop.RelType = path[i+1].Edge.RelType
			hop.Role = path[i+1].Edge.Role
		}
		result.Path = append(result.Path, hop)
	}

	return result
}

func pathPersonName(archive *glxlib.GLXFile, personID string) string {
	person, ok := archive.Persons[personID]
	if !ok || person == nil {
		return personID
	}
	// extractPersonName returns "(unnamed)" when no name property exists,
	// so fall back to the person ID for a cleaner display.
	names := extractAllNames(person)
	if len(names) == 0 {
		return personID
	}
	return names[0]
}

// printPathText prints the path in a human-readable format.
func printPathText(result *pathResult) {
	if result.Message != "" {
		fmt.Printf("\n  %s between %s and %s\n\n", result.Message, result.From, result.To)
		return
	}

	fmt.Printf("\nPath from %s to %s (%d hop(s)):\n\n", result.From, result.To, result.Hops)

	for _, hop := range result.Path {
		fmt.Printf("  %s (%s)\n", hop.PersonName, hop.PersonID)
		if hop.RelType != "" {
			relLabel := formatRelLabel(hop.RelType, hop.Role)
			fmt.Printf("    - %s ->\n", relLabel)
		}
	}

	fmt.Println()
}

// formatRelLabel creates a display label from relationship type and role.
func formatRelLabel(relType, role string) string {
	relType = strings.ReplaceAll(relType, "_", " ")
	if role != "" {
		return role + " in " + relType
	}
	return relType
}

// printPathJSON outputs the result as JSON.
func printPathJSON(result *pathResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
