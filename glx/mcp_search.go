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
	"unicode/utf8"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

const (
	mcpSearchMaxQueryRunes  = 1024
	mcpSearchMaxResultBytes = 256 * 1024
	// Reserve more than the fixed pagination wrapper, including integer fields.
	mcpSearchMatchBytes = mcpSearchMaxResultBytes - 256
)

var (
	errMCPSearchQueryLength = errors.New("search query must contain at most 1024 Unicode characters")
	errMCPSearchMatchSize   = errors.New("search match ID or field path exceeds the 256 KiB result limit")
)

type mcpSearchMatch struct {
	EntityType glxlib.EntityType `json:"entity_type"`
	EntityID   string            `json:"entity_id"`
	Field      string            `json:"field"`
	Value      string            `json:"value"`
}

type mcpSearchResult struct {
	Matches  []mcpSearchMatch `json:"matches"`
	Count    int              `json:"count"`
	Offset   int              `json:"offset"`
	Returned int              `json:"returned"`
	HasMore  bool             `json:"has_more"`
}

func (b *mcpArchiveBackend) searchContext(ctx context.Context, args *glxmcp.SearchArgs) (any, error) {
	query, entityType, err := normalizeSearchInput(args.Query, args.EntityType)
	if err != nil {
		if errors.Is(err, errUnknownSearchType) {
			// The CLI includes the supplied collection in its error. Avoid an
			// unbounded protocol error for a large or heavily escaped value.
			return nil, fmt.Errorf("%w: use list_entity_types for supported collections", errUnknownSearchType)
		}

		return nil, err
	}
	if utf8.RuneCountInString(query) > mcpSearchMaxQueryRunes {
		return nil, errMCPSearchQueryLength
	}
	if args.Offset < 0 || args.Limit < 0 || args.Limit > 1000 {
		return nil, errMCPPagination
	}
	limit := args.Limit
	if limit == 0 {
		limit = 100
	}
	page := mcpSearchPage{
		result: &mcpSearchResult{Matches: []mcpSearchMatch{}, Offset: args.Offset},
		limit:  limit, bytes: 2, // JSON array brackets; each row also reserves a comma.
	}
	if err := visitSearchArchiveContext(ctx, b.archive, query, args.CaseSensitive, entityType, page.visit); err != nil {
		return nil, err
	}
	if page.err != nil {
		return nil, page.err
	}
	result := page.result
	result.Returned = len(result.Matches)
	// Subtraction avoids overflow for arbitrarily large, valid offsets.
	result.HasMore = result.Count > args.Offset && result.Count-args.Offset > result.Returned

	return result, nil
}

type mcpSearchPage struct {
	result       *mcpSearchResult
	limit        int
	bytes        int
	full         bool
	previousType glxlib.EntityType
	previousID   string
	err          error
}

func (p *mcpSearchPage) visit(match searchResult) bool {
	// Each entity's fields are contiguous in the shared scanner. Keep the
	// same first match as CLI deduplication, retaining only the requested
	// page rather than all matching entities or a full-archive seen set.
	if p.result.Count > 0 && p.previousType == match.EntityType && p.previousID == match.EntityID {
		return true
	}
	p.previousType, p.previousID = match.EntityType, match.EntityID
	p.result.Count++
	if p.result.Count <= p.result.Offset || len(p.result.Matches) >= p.limit || p.full {
		return true
	}
	p.err = p.appendMatch(match)

	return p.err == nil
}

func (p *mcpSearchPage) appendMatch(match searchResult) error {
	// Reject oversized opaque references without truncating them or
	// allocating an unbounded JSON row. Ordinary previews are bounded
	// independently, including fields the CLI displays in full.
	if len(match.EntityID) > mcpSearchMatchBytes || len(match.Field) > mcpSearchMatchBytes {
		return errMCPSearchMatchSize
	}
	row := mcpSearchMatch{
		EntityType: match.EntityType, EntityID: match.EntityID,
		Field: match.Field, Value: truncate(match.Value),
	}
	encoded, err := json.Marshal(row)
	if err != nil {
		return err
	}
	if len(encoded)+3 > mcpSearchMatchBytes {
		return errMCPSearchMatchSize
	}
	if p.bytes+len(encoded)+1 > mcpSearchMatchBytes {
		p.full = true

		return nil
	}
	p.result.Matches = append(p.result.Matches, row)
	p.bytes += len(encoded) + 1

	return nil
}
