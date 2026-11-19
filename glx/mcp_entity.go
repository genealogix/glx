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
	"fmt"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

type mcpEntityResult struct {
	EntityType glxlib.EntityType `json:"entity_type"`
	ID         string            `json:"id"`
	Entity     json.RawMessage   `json:"entity"`
}

func (b *mcpArchiveBackend) getEntityContext(ctx context.Context, args *glxmcp.GetEntityArgs) (*mcpEntityResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(args.ID) == "" {
		return nil, errMCPEntityID
	}
	entity, err := mcpEntityByID(b.archive, args.EntityType, args.ID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("%w %q: %q", errMCPEntityNotFound, args.EntityType, args.ID)
	}
	// Use the same archive-field normalization as query, inside serialized
	// dispatch. No inference from ID prefixes and no filesystem lookup occur.
	encoded, err := marshalArchiveJSON(entity)
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if err != nil {
		return nil, err
	}

	return &mcpEntityResult{EntityType: args.EntityType, ID: args.ID, Entity: encoded}, nil
}

func mcpEntityByID(archive *glxlib.GLXFile, entityType glxlib.EntityType, id string) (any, error) {
	switch entityType {
	case glxlib.EntityTypePersons:
		return mcpEntityLookup(archive.Persons, id), nil
	case glxlib.EntityTypeEvents:
		return mcpEntityLookup(archive.Events, id), nil
	case glxlib.EntityTypeRelationships:
		return mcpEntityLookup(archive.Relationships, id), nil
	case glxlib.EntityTypePlaces:
		return mcpEntityLookup(archive.Places, id), nil
	case glxlib.EntityTypeSources:
		return mcpEntityLookup(archive.Sources, id), nil
	case glxlib.EntityTypeCitations:
		return mcpEntityLookup(archive.Citations, id), nil
	case glxlib.EntityTypeRepositories:
		return mcpEntityLookup(archive.Repositories, id), nil
	case glxlib.EntityTypeMedia:
		return mcpEntityLookup(archive.Media, id), nil
	case glxlib.EntityTypeAssertions:
		return mcpEntityLookup(archive.Assertions, id), nil
	case glxlib.EntityTypeResearchLogs:
		return mcpEntityLookup(archive.ResearchLogs, id), nil
	case glxlib.EntityTypeStudies:
		return mcpEntityLookup(archive.Studies, id), nil
	default:
		return nil, fmt.Errorf("%w: %q", errUnknownJSONEntityType, entityType)
	}
}

func mcpEntityLookup[T any](entities map[string]*T, id string) any {
	if entity := entities[id]; entity != nil {
		return entity
	}

	return nil
}
