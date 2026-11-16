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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

var (
	errUnknownMCPOperation = errors.New("unknown GLX operation")
	errMCPPagination       = errors.New("offset must be nonnegative and limit must be between 0 and 1000")
	errMCPEndpoints        = errors.New("from and to must identify distinct people")
	errMCPMaxHops          = errors.New("max_hops must be at least 1")
	errMCPEntityID         = errors.New("id must not be empty or whitespace")
	errMCPEntityNotFound   = errors.New("entity not found in collection")
)

// mcpArchiveBackend owns one snapshot. The protocol adapter serializes its
// calls, including the lazy validation cache; no handler reopens files. A watcher
// replaces the whole backend rather than mutating a published archive.
type mcpArchiveBackend struct {
	archive    *glxlib.GLXFile
	duplicates []string
	validation *validationJSONResult
}

func serveMCP(ctx context.Context, archivePath string, watch bool) error {
	var backend glxmcp.Backend
	var watcher *mcpWatchingBackend
	var err error
	if watch {
		watcher, err = newMCPWatchingBackend(ctx, archivePath)
		backend = watcher
	} else {
		backend, err = loadMCPArchiveContext(ctx, archivePath)
	}
	if err != nil {
		return err
	}
	server, err := glxmcp.NewServer(backend, versionString())
	if err != nil {
		return err
	}

	if watcher != nil {
		watchCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			watcher.run(watchCtx, SystemIOStreams())
		}()
		defer func() {
			cancel()
			<-done
		}()
	}

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}

func loadMCPArchive(path string) (*mcpArchiveBackend, error) {
	return loadMCPArchiveContext(context.Background(), path)
}

func loadMCPArchiveContext(ctx context.Context, path string) (*mcpArchiveBackend, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access archive: %w", err)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return nil, errMCPWatchSourceType
	}
	var archive *glxlib.GLXFile
	var duplicates []string
	if info.IsDir() {
		// Reading a fresh cache is permitted, but serving never creates or
		// rewrites a cache even when GLX_CACHE=auto is set in the environment.
		var cached bool
		if cacheEnvMode() != cacheModeOff {
			archive, duplicates, cached = tryLoadFreshCache(path)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !cached {
			// Use the same contained, regular-file-only reader as watch mode.
			// A FIFO present initially or swapped in after stat cannot block startup.
			source := mcpArchiveSource{path: path, directory: true}
			var files map[string][]byte
			files, err = source.captureDirectory(ctx)
			if err == nil {
				archive, duplicates, err = loadArchiveFromFiles(path, files, false)
			}
		}
	} else {
		var data []byte
		data, err = readMCPWatchSingleFile(path)
		if canceled := ctx.Err(); canceled != nil {
			return nil, canceled
		}
		if err == nil {
			archive, err = createSerializer(false, false, "").DeserializeSingleFileBytes(data)
		}
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := mergeStandardVocabularies(archive); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, duplicate := range duplicates {
		SystemIOStreams().Errorf("Warning: %s\n", duplicate)
	}

	return &mcpArchiveBackend{archive: archive, duplicates: duplicates}, nil
}

func (b *mcpArchiveBackend) CallTool(ctx context.Context, name string, raw json.RawMessage) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	switch name {
	case "query":
		return withMCPArgs(raw, func(a *glxmcp.QueryArgs) (any, error) { return b.queryContext(ctx, a) })
	case "search":
		return withMCPArgs(raw, func(a *glxmcp.SearchArgs) (any, error) { return b.searchContext(ctx, a) })
	case "get_entity":
		return withMCPArgs(raw, func(a *glxmcp.GetEntityArgs) (any, error) { return b.getEntityContext(ctx, a) })
	case "vitals":
		return withMCPArgs(raw, func(a *glxmcp.PersonArgs) (any, error) { return buildVitalsJSON(b.archive, a.Person) })
	case "timeline":
		//nolint:contextcheck // Legacy timeline collection is synchronous; the serialized adapter checks cancellation after it finishes.
		return withMCPArgs(raw, func(a *glxmcp.TimelineArgs) (any, error) { return buildTimelineJSON(b.archive, a.Person, !a.NoFamily) })
	case "summary":
		return withMCPArgs(raw, func(a *glxmcp.SummaryArgs) (any, error) {
			return buildSummaryJSONWithLogsContext(ctx, b.archive, a.Person, a.IncludeResearchLogs)
		})
	case jsonAncestors, jsonDescendants:
		return withMCPArgs(raw, func(a *glxmcp.TreeArgs) (any, error) {
			return b.tree(ctx, *a, name == jsonAncestors)
		})
	case "cite":
		return withMCPArgs(raw, func(a *glxmcp.CiteArgs) (any, error) { return buildCitationJSON(b.archive, a.Citation, a.All) })
	case "analyze":
		return withMCPArgs(raw, func(a *glxmcp.AnalyzeArgs) (any, error) { return b.analyzeContext(ctx, a) })
	case "stats":
		return buildStatsJSONContext(ctx, b.archive)
	case "places":
		return buildPlacesJSON(b.archive)
	case "path":
		return withMCPArgs(raw, func(a *glxmcp.PathArgs) (any, error) { return b.pathContext(ctx, a) })
	case "cluster":
		return withMCPArgs(raw, func(a *glxmcp.ClusterArgs) (any, error) { return b.clusterContext(ctx, a) })
	case "validate":
		return withMCPArgs(raw, func(a *glxmcp.ValidateArgs) (any, error) { return b.validateContext(ctx, a) })
	case "list_entity_types":
		return map[string]any{"entity_counts": archiveEntityCounts(b.archive)}, nil
	case "vocabularies":
		return mcpVocabularies(b.archive)
	default:
		return nil, fmt.Errorf("%w %q", errUnknownMCPOperation, name)
	}
}

func withMCPArgs[T any](raw json.RawMessage, call func(*T) (any, error)) (any, error) {
	var args T
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}

	return call(&args)
}

type mcpQueryResult struct {
	*queryJSONResult
	Offset   int  `json:"offset"`
	Returned int  `json:"returned"`
	HasMore  bool `json:"has_more"`
}

func (b *mcpArchiveBackend) queryContext(ctx context.Context, a *glxmcp.QueryArgs) (any, error) {
	if a.Offset < 0 || a.Limit < 0 || a.Limit > 1000 {
		return nil, errMCPPagination
	}
	if a.Limit == 0 {
		a.Limit = 100
	}
	result, err := buildQueryJSONContext(ctx, b.archive, glxlib.EntityType(a.EntityType), &queryOpts{
		Name: a.Name, Phonetic: a.Phonetic, BornBefore: a.BornBefore, BornAfter: a.BornAfter,
		Type: a.Type, Before: a.Before, After: a.After, Confidence: a.Confidence,
		Status: a.Status, Source: a.Source, Citation: a.Citation, Subject: a.Subject, Birthplace: a.Birthplace,
	})
	if err != nil {
		return nil, err
	}
	start := min(a.Offset, len(result.Entities))
	end := start + min(a.Limit, len(result.Entities)-start)
	result.Entities = result.Entities[start:end]
	// Normalize only the returned page while still inside serialized dispatch.
	// Encoding each entity is a cancellation boundary rather than deferring an
	// entire page's expensive YAML round trips to the SDK's output marshaler.
	for i := range result.Entities {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		encoded, err := marshalArchiveJSON(result.Entities[i].Entity)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result.Entities[i].encoded = encoded
	}

	return &mcpQueryResult{queryJSONResult: result, Offset: a.Offset, Returned: end - start, HasMore: end < result.Count}, nil
}

func (b *mcpArchiveBackend) analyzeContext(ctx context.Context, a *glxmcp.AnalyzeArgs) (any, error) {
	if a.Person != "" {
		id, _, err := resolveJSONPerson(b.archive, a.Person)
		if err != nil {
			return nil, err
		}
		a.Person = id
	}

	return buildAnalysisJSONContext(ctx, b.archive, a.Person, a.Check)
}

func (b *mcpArchiveBackend) pathContext(ctx context.Context, a *glxmcp.PathArgs) (any, error) {
	if strings.TrimSpace(a.From) == "" || strings.TrimSpace(a.To) == "" {
		return nil, errMCPEndpoints
	}
	if a.MaxHops == 0 {
		a.MaxHops = 10
	}
	if a.MaxHops < 1 {
		return nil, errMCPMaxHops
	}
	fromID, _, err := resolveJSONPerson(b.archive, a.From)
	if err != nil {
		return nil, fmt.Errorf("from person: %w", err)
	}
	toID, _, err := resolveJSONPerson(b.archive, a.To)
	if err != nil {
		return nil, fmt.Errorf("to person: %w", err)
	}
	if fromID == toID {
		return nil, fmt.Errorf("%w: both resolve to %s", errMCPEndpoints, fromID)
	}
	adj, err := buildPathAdjacencyContext(ctx, b.archive)
	if err != nil {
		return nil, err
	}
	if a.AcceptedOnly {
		adj = acceptedPathAdjacency(adj)
	}

	path, err := bfsPathContext(ctx, fromID, toID, adj, a.MaxHops)
	if err != nil {
		return nil, err
	}

	return buildPathResult(fromID, toID, path, b.archive), ctx.Err()
}

func (b *mcpArchiveBackend) clusterContext(ctx context.Context, a *glxmcp.ClusterArgs) (any, error) {
	id, _, err := resolveJSONPerson(b.archive, a.Person)
	if err != nil {
		return nil, err
	}

	return buildClusterContext(ctx, id, b.archive, a.Place, a.Before, a.After)
}

func mcpVocabularies(archive *glxlib.GLXFile) (map[string]any, error) {
	// Marshal a view containing only controlled vocabulary definitions; no
	// private genealogy entities or metadata are included in this resource.
	view := *archive
	view.ImportMetadata = nil
	view.Persons, view.Events, view.Relationships, view.Places = nil, nil, nil, nil
	view.Sources, view.Citations, view.Repositories, view.Media = nil, nil, nil, nil
	view.Assertions, view.ResearchLogs, view.Studies = nil, nil, nil
	data, err := yaml.Marshal(&view)
	if err != nil {
		return nil, err
	}

	return ParseYAMLFile(data)
}

func (b *mcpArchiveBackend) validateContext(ctx context.Context, a *glxmcp.ValidateArgs) (any, error) {
	if b.validation == nil {
		result, err := buildValidationJSONContext(ctx, b.archive, true)
		if err != nil {
			return nil, err
		}
		result.DuplicateIDs = append(result.DuplicateIDs, b.duplicates...)
		result.Valid = result.Valid && len(result.DuplicateIDs) == 0
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b.validation = result
	}
	result := *b.validation
	if !a.Report {
		result.Report = nil
	}

	return &result, nil
}
