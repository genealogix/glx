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

package lib

import (
	"regexp"
	"strings"
)

// PersonName represents parsed name components
type PersonName struct {
	Prefix        string // Mr., Dr., etc.
	GivenName     string // First/middle names
	Nickname      string // Nickname in quotes
	SurnamePrefix string // von, van, de, etc.
	Surname       string // Family name
	Suffix        string // Jr., Sr., III, etc.
}

// NameSubstructure holds NAME subrecord fields from GEDCOM
type NameSubstructure struct {
	NPFX string // Name prefix
	GIVN string // Given name
	NICK string // Nickname
	SPFX string // Surname prefix
	SURN string // Surname
	NSFX string // Name suffix
}

// parseGEDCOMName parses a GEDCOM name string into components
// Format: "Given /Surname/" or with substructure fields
func parseGEDCOMName(nameValue string, substructure *NameSubstructure) PersonName {
	name := PersonName{}

	if nameValue == "" {
		return name
	}

	// Extract surname from /surname/ notation
	surnameRegex := regexp.MustCompile(`/([^/]+)/`)
	matches := surnameRegex.FindStringSubmatch(nameValue)
	if len(matches) > 1 {
		name.Surname = strings.TrimSpace(matches[1])
	}

	// Remove surname from name value to get other parts
	nameWithoutSurname := surnameRegex.ReplaceAllString(nameValue, " ")
	nameWithoutSurname = strings.TrimSpace(nameWithoutSurname)

	// Extract nickname from "nickname" notation
	nicknameRegex := regexp.MustCompile(`"([^"]+)"`)
	nickMatches := nicknameRegex.FindStringSubmatch(nameWithoutSurname)
	if len(nickMatches) > 1 {
		name.Nickname = strings.TrimSpace(nickMatches[1])
		nameWithoutSurname = nicknameRegex.ReplaceAllString(nameWithoutSurname, " ")
		nameWithoutSurname = strings.TrimSpace(nameWithoutSurname)
	}

	// Parse remaining parts
	if nameWithoutSurname != "" {
		parts := strings.Fields(nameWithoutSurname)
		var givenParts []string

		for i, part := range parts {
			switch {
			case i == 0 && isNamePrefix(part):
				name.Prefix = part
			case i == len(parts)-1 && isNameSuffix(part):
				name.Suffix = part
			default:
				givenParts = append(givenParts, part)
			}
		}

		name.GivenName = strings.Join(givenParts, " ")
	}

	// Override with substructure if provided
	if substructure != nil {
		if substructure.NPFX != "" {
			name.Prefix = substructure.NPFX
		}
		if substructure.GIVN != "" {
			name.GivenName = substructure.GIVN
		}
		if substructure.NICK != "" {
			name.Nickname = substructure.NICK
		}
		if substructure.SPFX != "" {
			name.SurnamePrefix = substructure.SPFX
		}
		if substructure.SURN != "" {
			name.Surname = substructure.SURN
		}
		if substructure.NSFX != "" {
			name.Suffix = substructure.NSFX
		}
	}

	// Check for surname prefix
	if name.Surname != "" {
		surnameParts := strings.Fields(name.Surname)
		if len(surnameParts) > 1 && isSurnamePrefix(surnameParts[0]) {
			name.SurnamePrefix = surnameParts[0]
			name.Surname = strings.Join(surnameParts[1:], " ")
		}
	}

	return name
}

// isSurnamePrefix checks if a word is a surname prefix
func isSurnamePrefix(word string) bool {
	prefixes := map[string]bool{
		"von": true, "van": true, "de": true, "der": true, "den": true,
		"del": true, "della": true, "di": true, "da": true, "le": true,
		"la": true, "du": true, "des": true, "af": true, "av": true,
	}

	return prefixes[strings.ToLower(word)]
}

// isNamePrefix checks if a word is a name prefix
func isNamePrefix(word string) bool {
	prefixes := map[string]bool{
		"Mr.": true, "Mrs.": true, "Ms.": true, "Miss": true, "Dr.": true,
		"Prof.": true, "Rev.": true, "Hon.": true, "Sir": true, "Lady": true,
		"Lord": true, "Count": true, "Duke": true, "Baron": true,
	}

	return prefixes[word]
}

// isNameSuffix checks if a word is a name suffix
func isNameSuffix(word string) bool {
	suffixes := map[string]bool{
		"Jr.": true, "Jr": true, "Sr.": true, "Sr": true,
		"II": true, "III": true, "IV": true, "V": true,
		"2nd": true, "3rd": true, "4th": true,
		"Esq.": true, "Esq": true, "PhD": true, "MD": true,
	}

	return suffixes[word]
}

// FormatFullName constructs a display string from the PersonName components.
// The format follows: Prefix Given "Nickname" SurnamePrefix Surname Suffix
func (n PersonName) FormatFullName() string {
	var parts []string

	if n.Prefix != "" {
		parts = append(parts, n.Prefix)
	}
	if n.GivenName != "" {
		parts = append(parts, n.GivenName)
	}
	if n.Nickname != "" {
		parts = append(parts, "\""+n.Nickname+"\"")
	}
	if n.SurnamePrefix != "" {
		parts = append(parts, n.SurnamePrefix)
	}
	if n.Surname != "" {
		parts = append(parts, n.Surname)
	}
	if n.Suffix != "" {
		parts = append(parts, n.Suffix)
	}

	return strings.Join(parts, " ")
}

// ToFields converts a NameSubstructure to a map suitable for the name property's fields.
// Only includes fields that were explicitly present in GEDCOM substructure tags.
// We do NOT infer fields from parsing the name string - only explicit GEDCOM tags are used.
func (ns *NameSubstructure) ToFields() map[string]string {
	if ns == nil {
		return nil
	}

	fields := make(map[string]string)

	if ns.NPFX != "" {
		fields[NameFieldPrefix] = ns.NPFX
	}
	if ns.GIVN != "" {
		fields[NameFieldGiven] = ns.GIVN
	}
	if ns.NICK != "" {
		fields[NameFieldNickname] = ns.NICK
	}
	if ns.SPFX != "" {
		fields[NameFieldSurnamePrefix] = ns.SPFX
	}
	if ns.SURN != "" {
		fields[NameFieldSurname] = ns.SURN
	}
	if ns.NSFX != "" {
		fields[NameFieldSuffix] = ns.NSFX
	}

	if len(fields) == 0 {
		return nil
	}

	return fields
}

// ExtractNameFields extracts the given and surname from a person's name property.
// Returns empty strings if the fields are not found.
func ExtractNameFields(nameProperty any) (given, surname string) {
	if nameProperty == nil {
		return "", ""
	}

	// Handle map[string]any (the typical structure from YAML parsing or code)
	if nameMap, ok := nameProperty.(map[string]any); ok {
		// Check for fields sub-map
		if fieldsVal, hasFields := nameMap["fields"]; hasFields {
			if fields, ok := fieldsVal.(map[string]any); ok {
				if g, ok := fields[NameFieldGiven].(string); ok {
					given = g
				}
				if s, ok := fields[NameFieldSurname].(string); ok {
					surname = s
				}
			} else if fields, ok := fieldsVal.(map[string]string); ok {
				given = fields[NameFieldGiven]
				surname = fields[NameFieldSurname]
			}
		}
	}

	return given, surname
}

// GetFullName extracts the full name value from a person's name property.
// Returns empty string if not found.
func GetFullName(nameProperty any) string {
	if nameProperty == nil {
		return ""
	}

	// Handle simple string
	if s, ok := nameProperty.(string); ok {
		return s
	}

	// Handle map[string]any
	if nameMap, ok := nameProperty.(map[string]any); ok {
		if value, ok := nameMap["value"].(string); ok {
			return value
		}
	}

	return ""
}
