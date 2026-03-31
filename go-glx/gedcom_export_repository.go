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

// exportRepository converts a GLX Repository to a GEDCOM REPO record.
func exportRepository(repoID string, repo *Repository, expCtx *ExportContext) *GEDCOMRecord {
	xref := expCtx.RepositoryXRefMap[repoID]

	record := &GEDCOMRecord{
		XRef:       xref,
		Tag:        GedcomTagRepo,
		SubRecords: []*GEDCOMRecord{},
	}

	// NAME
	if repo.Name != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagName,
			Value: repo.Name,
		})
	}

	// ADDR with structured subrecords
	if repo.Address != "" || repo.City != "" || repo.State != "" || repo.PostalCode != "" || repo.Country != "" {
		addrRecord := &GEDCOMRecord{
			Tag:        GedcomTagAddr,
			SubRecords: []*GEDCOMRecord{},
		}

		// Main address line
		if repo.Address != "" {
			addrRecord.SubRecords = append(addrRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagAdr1,
				Value: repo.Address,
			})
		}

		if repo.City != "" {
			addrRecord.SubRecords = append(addrRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagCity,
				Value: repo.City,
			})
		}

		if repo.State != "" {
			addrRecord.SubRecords = append(addrRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagStae,
				Value: repo.State,
			})
		}

		if repo.PostalCode != "" {
			addrRecord.SubRecords = append(addrRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagPost,
				Value: repo.PostalCode,
			})
		}

		if repo.Country != "" {
			addrRecord.SubRecords = append(addrRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagCtry,
				Value: repo.Country,
			})
		}

		record.SubRecords = append(record.SubRecords, addrRecord)
	}

	// Phones from properties
	if phones, ok := getStringSliceProperty(repo.Properties, "phones"); ok {
		for _, phone := range phones {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagPhon,
				Value: phone,
			})
		}
	}

	// Emails from properties
	if emails, ok := getStringSliceProperty(repo.Properties, "emails"); ok {
		for _, email := range emails {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagEmail,
				Value: email,
			})
		}
	}

	// Website
	if repo.Website != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagWww,
			Value: repo.Website,
		})
	}

	// Notes
	for _, note := range repo.Notes {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: note,
		})
	}

	return record
}

// getStringSliceProperty extracts a string slice from a properties map.
// Handles both []string and []any (from YAML unmarshaling) types.
func getStringSliceProperty(props map[string]any, key string) ([]string, bool) {
	val, ok := props[key]
	if !ok {
		return nil, false
	}

	// Try []string first
	if ss, ok := val.([]string); ok {
		return ss, true
	}

	// Try []any (common from YAML unmarshaling)
	if sa, ok := val.([]any); ok {
		result := make([]string, 0, len(sa))
		for _, v := range sa {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result, true
		}
	}

	return nil, false
}
