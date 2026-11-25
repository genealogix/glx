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

package mcp

import (
	glxlib "github.com/genealogix/glx/go-glx"
)

// GetEntityArgs selects one exact ID within a required entity collection.
// IDs are opaque keys, never file paths or person-name queries.
type GetEntityArgs struct {
	EntityType glxlib.EntityType `json:"entity_type" jsonschema:"Exact collection: persons, events, relationships, places, sources, citations, repositories, media, assertions, research_logs, or studies"`
	ID         string            `json:"id"          jsonschema:"Exact case-sensitive entity ID in that collection; no name matching or file access"`
}

// QueryArgs selects an entity collection and the same filters as glx query.
// Limit and Offset paginate the sorted result; zero Limit uses 100 rows.
type QueryArgs struct {
	EntityType string `json:"entity_type"           jsonschema:"Entity collection: persons, events, relationships, places, sources, citations, repositories, media, assertions, research_logs, or studies"`
	Name       string `json:"name,omitempty"        jsonschema:"Case-insensitive name/title substring for persons, sources, places, or repositories only; use get_entity for a known ID in any collection"`
	Phonetic   bool   `json:"phonetic,omitempty"    jsonschema:"Phonetic person-name matching; requires name"`
	BornBefore int    `json:"born_before,omitempty"`
	BornAfter  int    `json:"born_after,omitempty"`
	Type       string `json:"type,omitempty"        jsonschema:"Event, source, or relationship type"`
	Before     int    `json:"before,omitempty"      jsonschema:"Event year upper bound"`
	After      int    `json:"after,omitempty"       jsonschema:"Event year lower bound"`
	Confidence string `json:"confidence,omitempty"`
	Status     string `json:"status,omitempty"`
	Source     string `json:"source,omitempty"`
	Citation   string `json:"citation,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Birthplace string `json:"birthplace,omitempty"`
	Limit      int    `json:"limit,omitempty"       jsonschema:"Maximum rows, 1 to 1000; omitted or zero means 100"`
	Offset     int    `json:"offset,omitempty"      jsonschema:"Number of sorted matches to skip, at least zero"`
}

// SearchArgs searches recorded text using the same matching rules as glx search.
// Pagination counts distinct entities, retaining the first matching field.
type SearchArgs struct {
	Query         string `json:"query"                    jsonschema:"Nonempty text substring, at most 1024 Unicode characters; surrounding whitespace is trimmed"`
	EntityType    string `json:"entity_type,omitempty"    jsonschema:"Optional collection: persons, events, relationships, places, sources, citations, repositories, media, assertions, research_logs, or studies; omitted searches all"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" jsonschema:"Use case-sensitive substring matching; default is case-insensitive"`
	Limit         int    `json:"limit,omitempty"          jsonschema:"Maximum entities, 1 to 1000; omitted or zero means 100; pages also capped at 256 KiB"`
	Offset        int    `json:"offset,omitempty"         jsonschema:"Number of matching entities to skip, at least zero"`
}

// PersonArgs identifies a person by exact ID or an unambiguous name substring.
type PersonArgs struct {
	Person string `json:"person" jsonschema:"Exact person ID or unambiguous case-insensitive name substring; use query to disambiguate"`
}

// SummaryArgs requests a profile with compact research log references by default.
type SummaryArgs struct {
	Person              string `json:"person"                          jsonschema:"Exact person ID or unambiguous name substring"`
	IncludeResearchLogs bool   `json:"include_research_logs,omitempty" jsonschema:"Include complete research logs in addition to compact references; can produce large responses. Use get_entity for one log instead"`
}

// TimelineArgs controls whether a person's timeline includes family events.
type TimelineArgs struct {
	Person   string `json:"person"`
	NoFamily bool   `json:"no_family,omitempty" jsonschema:"Exclude related family events"`
}

// TreeArgs controls an ancestor or descendant traversal.
type TreeArgs struct {
	Person      string `json:"person"`
	Generations int    `json:"generations,omitempty" jsonschema:"Maximum generations, 1 to 20; omitted or zero means 4; results are capped at 1000 nodes"`
}

// PathArgs selects endpoints and limits for a shortest relationship path.
type PathArgs struct {
	From         string `json:"from"`
	To           string `json:"to"`
	MaxHops      int    `json:"max_hops,omitempty"      jsonschema:"Maximum relationship hops; omitted or zero means 10"`
	AcceptedOnly bool   `json:"accepted_only,omitempty" jsonschema:"Exclude relationships recorded only as hypotheses"`
}

// CiteArgs selects one citation or every citation in the archive.
type CiteArgs struct {
	Citation string `json:"citation,omitempty" jsonschema:"Exact citation ID; required unless all is true"`
	All      bool   `json:"all,omitempty"`
}

// AnalyzeArgs selects a research analysis category and optional person scope.
type AnalyzeArgs struct {
	Person string `json:"person,omitempty"`
	Check  string `json:"check,omitempty"  jsonschema:"gaps, evidence, consistency, conflicts, or suggestions; omitted means every category"`
}

// ClusterArgs limits a person's FAN (friends, associates, neighbors) analysis.
type ClusterArgs struct {
	Person string `json:"person"`
	Place  string `json:"place,omitempty"`
	Before int    `json:"before,omitempty"`
	After  int    `json:"after,omitempty"`
}

// ValidateArgs optionally includes the assertion confidence report.
type ValidateArgs struct {
	Report bool `json:"report,omitempty"`
}
