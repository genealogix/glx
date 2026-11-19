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

package glx

import (
	"net/url"
	"slices"
	"strings"
)

// Known authority names accepted by glx add/link. URI inputs remain unchanged.
var externalIDAuthorities = map[string]string{
	"wikitree":     "https://www.wikitree.com/wiki/",
	"familysearch": "https://www.familysearch.org/tree/person/details/",
	"findagrave":   "https://www.findagrave.com/memorial/",
	"ancestry":     "https://www.ancestry.com/",
}

// externalIDOpaquePrefix wraps an authority name that is not a URI so it can
// travel in a GEDCOM 7 EXID.TYPE.
const externalIDOpaquePrefix = "urn:glx:external-id:"

// importExternalIDAuthority reverses the 7.0 export mapping: a known authority
// URI or an opaque GLX URN becomes the short authority name GLX stores, so
// identifiers keep one representation across export and re-import.
func importExternalIDAuthority(authority string) string {
	if name, ok := strings.CutPrefix(authority, externalIDOpaquePrefix); ok {
		if unescaped, err := url.PathUnescape(name); err == nil && unescaped != "" {
			return unescaped
		}

		return authority
	}
	for name, uri := range externalIDAuthorities {
		if authority == uri {
			return name
		}
	}

	return authority
}

// exportEntityExternalIDs exports every property the vocabulary maps to
// EXID, in key order, falling back to the standard external_ids key when the
// vocabulary maps none.
func exportEntityExternalIDs(props map[string]any, index map[string]string, version GEDCOMVersion) []*GEDCOMRecord {
	var keys []string
	for key, tag := range index {
		if tag == GedcomTagExid {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		keys = []string{"external_ids"}
	}
	slices.Sort(keys)
	var records []*GEDCOMRecord
	for _, key := range keys {
		if value, ok := props[key]; ok {
			records = append(records, exportExternalIDs(value, version)...)
		}
	}

	return records
}

func exportExternalIDs(value any, version GEDCOMVersion) []*GEDCOMRecord {
	var items []any
	switch entries := value.(type) {
	case []any:
		items = entries
	case []string:
		for _, entry := range entries {
			items = append(items, entry)
		}
	default:
		items = []any{value}
	}
	var records []*GEDCOMRecord
	for _, item := range items {
		var id, authority string
		switch entry := item.(type) {
		case string:
			id = entry
		case map[string]any:
			id, _ = entry["value"].(string)
			if fields, ok := entry["fields"].(map[string]any); ok {
				authority, _ = fields["type"].(string)
			}
		}
		if id == "" {
			continue
		}
		tag := GedcomTagRefn
		if version == GEDCOM70 {
			tag = GedcomTagExid
			if known := externalIDAuthorities[strings.ToLower(authority)]; known != "" {
				authority = known
			} else if parsed, err := url.Parse(authority); authority != "" && (err != nil || !parsed.IsAbs()) {
				// An opaque URI retains an unrecognized authority without inventing a website.
				authority = externalIDOpaquePrefix + url.PathEscape(authority)
			}
		}
		record := &GEDCOMRecord{Tag: tag, Value: id}
		if authority != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{Tag: GedcomTagType, Value: authority})
		}
		records = append(records, record)
	}

	return records
}
