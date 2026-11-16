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

// QueryArgs selects an entity collection and the same filters as glx query.
// Limit and Offset paginate the sorted result; zero Limit uses 100 rows.
type QueryArgs struct {
	EntityType string `json:"entity_type"           jsonschema:"Entity collection: persons, events, relationships, places, sources, citations, repositories, media, assertions, research_logs, or studies"`
	Name       string `json:"name,omitempty"        jsonschema:"Case-insensitive name or title substring"`
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

// PersonArgs identifies a person by exact ID or an unambiguous name substring.
type PersonArgs struct {
	Person string `json:"person" jsonschema:"Exact person ID or unambiguous case-insensitive name substring; use query to disambiguate"`
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
