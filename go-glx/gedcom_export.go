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
	"fmt"
	"io"
	"sort"
)

// ExportGEDCOM converts a GLX archive to GEDCOM format bytes.
// version selects GEDCOM551 or GEDCOM70.
// logWriter receives progress/diagnostic output (nil to suppress).
//
// If glx.EventTypes is nil, standard vocabularies are loaded into the GLXFile
// (mutating the input) to enable GEDCOM tag mapping.
func ExportGEDCOM(glx *GLXFile, version GEDCOMVersion, logWriter io.Writer) ([]byte, *ExportResult, error) {
	if glx == nil {
		return nil, nil, ErrGLXFileNil
	}

	logger := NewImportLogger(logWriter)
	logger.LogInfo("Starting GEDCOM export")

	// Load standard vocabularies if not already loaded
	if glx.EventTypes == nil {
		if err := LoadStandardVocabulariesIntoGLX(glx); err != nil {
			return nil, nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
		}
	}

	// Build export index (reverse of GEDCOM import index)
	exportIndex := buildExportIndex(glx)

	// Create export context
	expCtx := &ExportContext{
		GLX:               glx,
		Version:           version,
		Logger:            logger,
		ExportIndex:       exportIndex,
		PersonXRefMap:     make(map[string]string),
		SourceXRefMap:     make(map[string]string),
		RepositoryXRefMap: make(map[string]string),
		MediaXRefMap:      make(map[string]string),
		PlaceStrings:      make(map[string]string),
		Stats:             ExportStatistics{},
	}

	// Assign XREF IDs deterministically (sorted keys)
	assignXRefIDs(expCtx)

	// Pre-resolve all place strings
	resolvePlaceStrings(expCtx)

	// Build person events index (person ID -> event IDs where person is principal)
	buildPersonEventsIndex(expCtx)

	// Build assertion lookup index (person ID + property -> assertions)
	buildPersonPropertyAssertionsIndex(expCtx)

	// Reconstruct families from relationships (before building records)
	reconstructFamilies(expCtx)

	// Build GEDCOM records
	var records []*GEDCOMRecord

	// HEAD record
	records = append(records, buildHEADRecord(expCtx))

	// Repository records
	repoIDs := sortedKeys(glx.Repositories)
	for _, repoID := range repoIDs {
		repo := glx.Repositories[repoID]
		record := exportRepository(repoID, repo, expCtx)
		records = append(records, record)
		expCtx.Stats.RepositoriesExported++
	}

	// Source records (after repositories, since sources reference repos)
	sourceIDs := sortedKeys(glx.Sources)
	for _, sourceID := range sourceIDs {
		source := glx.Sources[sourceID]
		record := exportSource(sourceID, source, expCtx)
		records = append(records, record)
		expCtx.Stats.SourcesExported++
	}

	// Media records
	mediaIDs := sortedKeys(glx.Media)
	for _, mediaID := range mediaIDs {
		media := glx.Media[mediaID]
		record := exportMedia(mediaID, media, expCtx)
		records = append(records, record)
		expCtx.Stats.MediaExported++
	}

	// Person records
	personIDs := sortedKeys(glx.Persons)
	for _, personID := range personIDs {
		person := glx.Persons[personID]
		record := exportPerson(personID, person, expCtx)
		records = append(records, record)
		expCtx.Stats.PersonsExported++
	}

	// Family records
	for _, family := range expCtx.Families {
		record := exportFamily(family, expCtx)
		records = append(records, record)
		expCtx.Stats.FamiliesExported++
	}

	// SUBM record (required by GEDCOM 5.5.1)
	if expCtx.Version == GEDCOM551 {
		records = append(records, buildSUBMRecord(expCtx))
	}

	// TRLR record
	records = append(records, &GEDCOMRecord{Tag: GedcomTagTrlr})

	logger.LogInfof("Export completed: %d persons, %d families, %d repositories, %d sources, %d media",
		expCtx.Stats.PersonsExported, expCtx.Stats.FamiliesExported, expCtx.Stats.RepositoriesExported, expCtx.Stats.SourcesExported, expCtx.Stats.MediaExported)

	// Serialize to bytes
	data := serializeGEDCOMRecords(records)

	// Build result
	versionStr := GEDCOMVersion551
	if version == GEDCOM70 {
		versionStr = GEDCOMVersion70
	}

	result := &ExportResult{
		Statistics: expCtx.Stats,
		Version:    versionStr,
	}

	return data, result, nil
}

// ExportContext holds state during GLX to GEDCOM export.
type ExportContext struct {
	GLX     *GLXFile
	Version GEDCOMVersion
	Logger  *ImportLogger // reuse existing logger

	// Reverse vocabulary lookup (GLX key -> GEDCOM tag)
	ExportIndex *ExportIndex

	// GLX ID -> GEDCOM XREF maps
	PersonXRefMap     map[string]string
	SourceXRefMap     map[string]string
	RepositoryXRefMap map[string]string
	MediaXRefMap      map[string]string

	// Place cache: placeID -> full GEDCOM place string
	PlaceStrings map[string]string

	// PersonEvents maps person ID -> event IDs where person is principal
	PersonEvents map[string][]string

	// Reconstructed family records
	Families     []*ExportFamily
	FamilyXRefMap map[string]string // relationship ID -> family XREF

	// Person-to-family reverse maps for FAMS/FAMC back-references
	PersonSpouseFamilies map[string][]string          // person ID -> family XRefs where spouse
	PersonChildFamilies  map[string][]childFamilyRef  // person ID -> family refs where child

	// PersonPropertyAssertions maps personID -> property -> assertions
	// Used to export SOUR on NAME, OCCU, RESI, etc. from assertion evidence
	PersonPropertyAssertions map[string]map[string][]*Assertion

	Stats ExportStatistics
}

// ExportIndex provides forward lookups from GLX keys to GEDCOM tags.
// This is the reverse of GEDCOMIndex (which maps GEDCOM tags to GLX keys).
type ExportIndex struct {
	EventTypes           map[string]string // "birth" -> "BIRT"
	PersonProperties     map[string]string
	EventProperties      map[string]string
	CitationProperties   map[string]string
	SourceProperties     map[string]string
	RepositoryProperties map[string]string
	MediaProperties      map[string]string
	RelationshipTypes    map[string]string // "marriage" -> "MARR"
}

// ExportResult contains statistics about the export.
type ExportResult struct {
	Statistics ExportStatistics
	Version    string
}

// ExportStatistics tracks export metrics.
type ExportStatistics struct {
	PersonsExported      int
	FamiliesExported     int
	SourcesExported      int
	RepositoriesExported int
	MediaExported        int
	EventsProcessed      int
	PlacesResolved       int
	Warnings             []ExportWarning
}

// ExportWarning represents a warning during export.
type ExportWarning struct {
	EntityType string
	EntityID   string
	Message    string
}

// buildExportIndex constructs forward lookup indices from vocabularies in the GLXFile.
// This is the reverse of buildGEDCOMIndex: maps GLX keys to GEDCOM tags.
func buildExportIndex(glx *GLXFile) *ExportIndex {
	index := &ExportIndex{
		EventTypes:           make(map[string]string),
		PersonProperties:     make(map[string]string),
		EventProperties:      make(map[string]string),
		CitationProperties:   make(map[string]string),
		SourceProperties:     make(map[string]string),
		RepositoryProperties: make(map[string]string),
		MediaProperties:      make(map[string]string),
		RelationshipTypes:    make(map[string]string),
	}

	// Build event type index: GLX key -> GEDCOM tag
	for key, eventType := range glx.EventTypes {
		if eventType.GEDCOM != "" {
			index.EventTypes[key] = eventType.GEDCOM
		}
	}

	// Build relationship type index: GLX key -> GEDCOM tag
	for key, relType := range glx.RelationshipTypes {
		if relType.GEDCOM != "" {
			index.RelationshipTypes[key] = relType.GEDCOM
		}
	}

	// Build property indices: GLX key -> GEDCOM tag
	for key, propDef := range glx.PersonProperties {
		if propDef.GEDCOM != "" {
			index.PersonProperties[key] = propDef.GEDCOM
		}
	}

	for key, propDef := range glx.EventProperties {
		if propDef.GEDCOM != "" {
			index.EventProperties[key] = propDef.GEDCOM
		}
	}

	for key, propDef := range glx.CitationProperties {
		if propDef.GEDCOM != "" {
			index.CitationProperties[key] = propDef.GEDCOM
		}
	}

	for key, propDef := range glx.SourceProperties {
		if propDef.GEDCOM != "" {
			index.SourceProperties[key] = propDef.GEDCOM
		}
	}

	for key, propDef := range glx.RepositoryProperties {
		if propDef.GEDCOM != "" {
			index.RepositoryProperties[key] = propDef.GEDCOM
		}
	}

	for key, propDef := range glx.MediaProperties {
		if propDef.GEDCOM != "" {
			index.MediaProperties[key] = propDef.GEDCOM
		}
	}

	return index
}

// assignXRefIDs assigns deterministic GEDCOM XREF IDs to all entities.
// Iterates sorted keys for reproducible output.
func assignXRefIDs(expCtx *ExportContext) {
	// Persons: @I1@, @I2@, ...
	counter := 1
	for _, id := range sortedKeys(expCtx.GLX.Persons) {
		expCtx.PersonXRefMap[id] = fmt.Sprintf("@I%d@", counter)
		counter++
	}

	// Sources: @S1@, @S2@, ...
	counter = 1
	for _, id := range sortedKeys(expCtx.GLX.Sources) {
		expCtx.SourceXRefMap[id] = fmt.Sprintf("@S%d@", counter)
		counter++
	}

	// Repositories: @R1@, @R2@, ...
	counter = 1
	for _, id := range sortedKeys(expCtx.GLX.Repositories) {
		expCtx.RepositoryXRefMap[id] = fmt.Sprintf("@R%d@", counter)
		counter++
	}

	// Media: @O1@, @O2@, ...
	counter = 1
	for _, id := range sortedKeys(expCtx.GLX.Media) {
		expCtx.MediaXRefMap[id] = fmt.Sprintf("@O%d@", counter)
		counter++
	}
}

// buildHEADRecord constructs the GEDCOM HEAD record.
func buildHEADRecord(expCtx *ExportContext) *GEDCOMRecord {
	head := &GEDCOMRecord{
		Tag:        GedcomTagHead,
		SubRecords: []*GEDCOMRecord{},
	}

	// Source system
	head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
		Tag:   GedcomTagSour,
		Value: "GLX",
		SubRecords: []*GEDCOMRecord{
			{Tag: GedcomTagName, Value: "Genealogix"},
			{Tag: GedcomTagVers, Value: "1.0"},
		},
	})

	// GEDCOM version
	var versionStr string
	if expCtx.Version == GEDCOM70 {
		versionStr = GEDCOMVersion70
	} else {
		versionStr = GEDCOMVersion551
	}

	gedcRecord := &GEDCOMRecord{
		Tag: GedcomTagGedc,
		SubRecords: []*GEDCOMRecord{
			{Tag: GedcomTagVers, Value: versionStr},
		},
	}

	// GEDCOM 5.5.1 includes FORM
	if expCtx.Version == GEDCOM551 {
		gedcRecord.SubRecords = append(gedcRecord.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagForm,
			Value: "LINEAGE-LINKED",
		})
	}

	head.SubRecords = append(head.SubRecords, gedcRecord)

	// Character set (5.5.1 only)
	if expCtx.Version == GEDCOM551 {
		head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagChar,
			Value: "UTF-8",
		})
	}

	// Submitter reference (GEDCOM 5.5.1 requires SUBM)
	if expCtx.Version == GEDCOM551 {
		head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagSubm,
			Value: "@SUBM@",
		})
	}

	// Import metadata roundtrip preservation
	if meta := expCtx.GLX.ImportMetadata; meta != nil {
		if meta.Language != "" {
			head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagLang,
				Value: meta.Language,
			})
		}
		if meta.SourceFile != "" {
			head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagFile,
				Value: meta.SourceFile,
			})
		}
		if meta.Copyright != "" {
			head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagCopr,
				Value: meta.Copyright,
			})
		}
		if meta.Notes != "" {
			head.SubRecords = append(head.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagNote,
				Value: meta.Notes,
			})
		}
	}

	return head
}

// buildSUBMRecord constructs the GEDCOM SUBM (submitter) record from GLX metadata.
func buildSUBMRecord(expCtx *ExportContext) *GEDCOMRecord {
	subm := &GEDCOMRecord{
		XRef:       "@SUBM@",
		Tag:        GedcomTagSubm,
		SubRecords: []*GEDCOMRecord{},
	}

	name := "GLX Export"
	if expCtx.GLX.ImportMetadata != nil && expCtx.GLX.ImportMetadata.Submitter != nil && expCtx.GLX.ImportMetadata.Submitter.Name != "" {
		name = expCtx.GLX.ImportMetadata.Submitter.Name
	}
	subm.SubRecords = append(subm.SubRecords, &GEDCOMRecord{
		Tag:   GedcomTagName,
		Value: name,
	})

	return subm
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// addExportWarning adds a warning to the export context.
func (expCtx *ExportContext) addExportWarning(entityType, entityID, message string) {
	expCtx.Stats.Warnings = append(expCtx.Stats.Warnings, ExportWarning{
		EntityType: entityType,
		EntityID:   entityID,
		Message:    message,
	})
}
