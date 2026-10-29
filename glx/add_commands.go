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
	"os"

	"github.com/spf13/cobra"
)

// addCmd is the parent for every `glx add <entity-type>` subcommand.
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create entities (person, place, event, …) from CLI flags",
	Long: `Create GLX entities from command-line flags instead of writing YAML by hand.

Subcommands:
  add person         Create a person
  add place          Create a place
  add event          Create an event
  add repository     Create a repository
  add source         Create a source
  add citation       Create a citation
  add relationship   Create a relationship
  add assertion      Create an assertion
  add research-log   Create a research log
  add search         Append one search to an existing research log
  add study          Create a study

Every subcommand validates supplied values against the archive's vocabularies
and entity references, and every date flag against the GLX date grammar,
before writing. The created entity ID is the only
thing written to stdout (progress goes to stderr), so it can be captured with
shell substitution (add search echoes the ID of the log it appended to):

    person_id=$(glx add person --given Johann --surname Jungk --archive .)
    glx add event --type christening --principal "$person_id" --archive .`,
	Args: cobra.NoArgs,
	RunE: showHelp,
}

// addStreams returns the streams every add subcommand writes through. The
// created ID goes to stdout (MachineOut) and nothing else does: progress
// lines go to stderr, so `id=$(glx add …)` captures exactly the ID with or
// without --quiet. --quiet still silences the progress lines.
func addStreams() *IOStreams {
	streams := SystemIOStreams()
	if !quietOutput {
		streams.Out = os.Stderr
	}

	return streams
}

// addCommonFlags wires the shared flags (archive path, --id, --force, etc.)
// onto a subcommand. dest must point to the embedded addCommonOptions of the
// per-subcommand options struct so flag values land in the right slot.
func addCommonFlags(cmd *cobra.Command, dest *addCommonOptions) {
	cmd.Flags().StringVarP(&dest.ArchivePath, "archive", "a", ".", "Archive path (directory)")
	cmd.Flags().StringVar(&dest.OverrideID, "id", "", "Override the derived entity ID")
	cmd.Flags().BoolVar(&dest.Force, "force", false, "Overwrite an existing entity with the chosen ID")
	cmd.Flags().BoolVar(&dest.SkipValidate, "skip-validate", false, "Skip whole-archive validation after adding (vocab and reference checks still run)")
	cmd.Flags().BoolVar(&dest.DryRun, "dry-run", false, "Print what would be created without writing files")
	// StringArrayVar (not StringSliceVar) — StringSliceVar would silently
	// split on commas, mangling a note like "John, born 1850" into two
	// separate notes. StringArrayVar treats each --note invocation as one
	// whole value. Applies to all repeatable string flags on add subcommands.
	cmd.Flags().StringArrayVar(&dest.Notes, "note", nil, "Free-text note (repeatable)")
}

// =============================================================================
// add person
// =============================================================================

var addPersonOpts addPersonOptions

var addPersonCmd = &cobra.Command{
	Use:   "person",
	Short: "Create a person entity",
	Long: `Create a Person entity in the archive.

At least one of --given, --surname, or --id must be supplied. The derived ID
is "person-GIVEN-SURNAME" with each part slug-lowercased; --id overrides it.

Sex (recorded sex, GEDCOM SEX) and gender (self-identified) are validated
against the archive's sex_types and gender_types vocabularies. Standard
vocabularies are loaded automatically for archives that have not customized
them.

--external-id is repeatable in the form type:value (e.g. wikitree:Little-20642)
and fills the person's external_ids property.`,
	Example: `  # Mint a person from given + surname
  glx add person --given "Johann Peter" --surname "Jungk" --sex male --archive ./archive

  # Capture the ID into a shell variable
  pid=$(glx add person --given Anna --surname Müller --archive ./archive)
  echo "created $pid"

  # Override the ID explicitly
  glx add person --id person-jungk-johann-peter --given Johann --surname Jungk --archive ./archive

  # Record identifiers from other systems
  glx add person --given Lewis --surname Little \
    --external-id wikitree:Little-20642 --external-id familysearch:LZX1-ABC --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddPerson,
}

func runAddPerson(_ *cobra.Command, _ []string) error {
	return addPerson(addStreams(), &addPersonOpts)
}

// =============================================================================
// add place
// =============================================================================

var (
	addPlaceOpts                   addPlaceOptions
	addPlaceLat, addPlaceLng       float64
	addPlaceLatSet, addPlaceLngSet bool
)

var addPlaceCmd = &cobra.Command{
	Use:   "place",
	Short: "Create a place entity",
	Long: `Create a Place entity in the archive.

--name is required (or --id). --parent must reference an existing place.
--type is validated against the place_types vocabulary.`,
	Example: `  # Top-level place (country)
  glx add place --name "Germany" --type country --archive ./archive

  # Place with a parent
  glx add place --name "Enkirch" --type town --parent place-rheinland-pfalz --archive ./archive

  # With coordinates
  glx add place --name "Enkirch" --type town --lat 49.97 --lng 7.13 --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddPlace,
}

func runAddPlace(_ *cobra.Command, _ []string) error {
	if addPlaceLatSet {
		addPlaceOpts.Latitude = &addPlaceLat
	}
	if addPlaceLngSet {
		addPlaceOpts.Longitude = &addPlaceLng
	}

	return addPlace(addStreams(), &addPlaceOpts)
}

// =============================================================================
// add event
// =============================================================================

var addEventOpts addEventOptions

var addEventCmd = &cobra.Command{
	Use:   "event",
	Short: "Create an event entity",
	Long: `Create an Event entity in the archive.

--type is required and is validated against the event_types vocabulary.
--principal is a shorthand for adding the named person with role "principal".
--participant is repeatable in the form person-id:role.
At least one of --principal or --participant is required.
--date must be a valid GLX date string (e.g. 1725-02-25, ABT 1850, JULIAN 1643-02-20).
--property is repeatable in the form key=value; each key must be defined in
the event_properties vocabulary (event_subtype, description, cause, ...).`,
	Example: `  # Christening with a principal and a place
  glx add event --type christening --date 1725-02-25 \
    --place place-enkirch \
    --principal person-johann-peter-jungk \
    --archive ./archive

  # Marriage with multiple participants
  glx add event --type marriage --date 1850 \
    --participant person-john:groom \
    --participant person-jane:bride \
    --archive ./archive

  # Event with properties
  glx add event --type death --date 1826 --principal person-lewis-little \
    --property cause="fever" --property description="Died intestate" \
    --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddEvent,
}

func runAddEvent(_ *cobra.Command, _ []string) error {
	return addEvent(addStreams(), &addEventOpts)
}

// =============================================================================
// add repository
// =============================================================================

var addRepoOpts addRepositoryOptions

var addRepoCmd = &cobra.Command{
	Use:   "repository",
	Short: "Create a repository entity",
	Long: `Create a Repository entity in the archive.

--name is required (or --id). --type is validated against the
repository_types vocabulary.`,
	Example: `  # Online database
  glx add repository --name "FamilySearch" --type database --website https://www.familysearch.org --archive ./archive

  # Physical archive
  glx add repository --name "Virginia State Library" --type library \
    --address "800 E. Broad St" --city "Richmond" --state "VA" --country "USA" --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddRepository,
}

func runAddRepository(_ *cobra.Command, _ []string) error {
	return addRepository(addStreams(), &addRepoOpts)
}

// =============================================================================
// add source
// =============================================================================

var addSourceOpts addSourceOptions

var addSourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Create a source entity",
	Long: `Create a Source entity in the archive.

--title is required (or --id). --type is validated against the source_types
vocabulary. --repository must reference an existing repository.

--url, --publication-info, --call-number, --source-nature and
--information-type set the matching source properties. --source-nature and
--information-type are validated against the source_natures and
information_types vocabularies.`,
	Example: `  # Indexed database source
  glx add source --title "Deutschland Geburten und Taufen, 1558-1898" \
    --type database --repository repository-familysearch --archive ./archive

  # Book with author(s) and date
  glx add source --title "Genealogy of the Smith Family" \
    --type book --author "Jane Smith" --date 1923 --archive ./archive

  # Online database with its evidence classification
  glx add source --title "Find a Grave Memorials" --type database \
    --url https://www.findagrave.com --source-nature derivative \
    --information-type secondary --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddSource,
}

func runAddSource(_ *cobra.Command, _ []string) error {
	return addSource(addStreams(), &addSourceOpts)
}

// =============================================================================
// add citation
// =============================================================================

var addCitationOpts addCitationOptions

var addCitationCmd = &cobra.Command{
	Use:   "citation",
	Short: "Create a citation entity",
	Long: `Create a Citation entity in the archive.

--source is required and must reference an existing source.
--external-id is repeatable in the form type:value (e.g.
familysearch:ark:/61903/1:1:C4H8-2DW2).`,
	Example: `  # Citation with URL and external ID
  # --external-id is parsed as TYPE:VALUE on the FIRST colon, so the
  # type cannot itself contain colons. Use a short identifier scheme name
  # (familysearch, wikidata, geonames, ...) as the type.
  glx add citation --source source-fs-births-deutschland \
    --url "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2" \
    --accessed 2026-04-20 \
    --external-id "familysearch:ark:/61903/1:1:C4H8-2DW2" \
    --archive ./archive

  # Citation with a locator (page or entry number) and transcription
  glx add citation --source source-1860-census \
    --locator "p. 47, dwelling 312" \
    --text-from-source "John Webb, 35, farmer …" \
    --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddCitation,
}

func runAddCitation(_ *cobra.Command, _ []string) error {
	return addCitation(addStreams(), &addCitationOpts)
}

// =============================================================================
// add relationship
// =============================================================================

var addRelOpts addRelationshipOptions

var addRelCmd = &cobra.Command{
	Use:   "relationship",
	Short: "Create a relationship entity",
	Long: `Create a Relationship entity in the archive.

--type is required and is validated against the relationship_types vocabulary.
At least two participants are required. Use --parent / --child (each
repeatable) for parent-child relationships, --participant person-id:role for
arbitrary roles. --start-event and --end-event must reference existing
events.`,
	Example: `  # Biological parent-child
  glx add relationship --type biological_parent_child \
    --parent person-johann --child person-johann-jr --archive ./archive

  # Marriage tied to a marriage event
  glx add relationship --type marriage \
    --participant person-john:spouse --participant person-jane:spouse \
    --start-event event-marriage-1850 \
    --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddRelationship,
}

func runAddRelationship(_ *cobra.Command, _ []string) error {
	return addRelationship(addStreams(), &addRelOpts)
}

// =============================================================================
// add assertion
// =============================================================================

var addAssertionOpts addAssertionOptions

var addAssertionCmd = &cobra.Command{
	Use:   "assertion",
	Short: "Create an assertion entity",
	Long: `Create an Assertion entity in the archive.

Exactly one of --subject-person, --subject-event, --subject-relationship,
--subject-place is required. The payload is either --property + --value (for
fact assertions) or --participant person-id:role (for participant assertions);
the two payload forms are mutually exclusive.

--confidence is validated against the confidence_levels vocabulary. --source,
--citation, and --media are each repeatable and must reference existing
entities.`,
	Example: `  # Birth-date assertion on an event
  glx add assertion --subject-event event-birth-johann \
    --property date --value 1725-02-18 \
    --citation citation-fs-jungk \
    --confidence medium \
    --archive ./archive

  # Participant assertion (a person played a role in an event)
  glx add assertion --subject-event event-christening-johann \
    --participant person-anna:godparent \
    --citation citation-fs-jungk \
    --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddAssertion,
}

func runAddAssertion(_ *cobra.Command, _ []string) error {
	return addAssertion(addStreams(), &addAssertionOpts)
}

// =============================================================================
// add research-log
// =============================================================================

var addResearchLogOpts addResearchLogOptions

var addResearchLogCmd = &cobra.Command{
	Use:   "research-log",
	Short: "Create a research log entity",
	Long: `Create a ResearchLog entity in the archive.

A research log records what was searched, where, and with what result,
including searches that found nothing (negative evidence). Create the log
once, then append searches to it with "glx add search".

At most one of --subject-person, --subject-event, --subject-relationship,
--subject-place may be given. --status is validated against the
research_log_status_types vocabulary. The derived ID is "research-log-" plus
the --title, else the subject, else the --objective; --id overrides it.
Repeated --citation IDs are included once, in the order first supplied.`,
	Example: `  # Open a log about a person
  glx add research-log --id rl-death --subject-person person-lewis-little \
    --objective "Verify the reported 1826 intestate death" --status in_progress \
    --archive ./archive

  # Log with a title, researcher and date
  glx add research-log --title "1860 census, Jane Webb" --researcher "I. Schepp" \
    --date 2026-09-17 --status open --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddResearchLog,
}

func runAddResearchLog(_ *cobra.Command, _ []string) error {
	return addResearchLog(addStreams(), &addResearchLogOpts)
}

// =============================================================================
// add search
// =============================================================================

var addSearchOpts addSearchOptions

var addSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Append a search to a research log",
	Long: `Append one search entry to an existing research log.

--log is required and must name an existing research log. At least one of
--source, --repository, --collection, --query or --citation says what was
searched. --result is validated against the search_result_types vocabulary
(found, not_found, inconclusive, partial, not_searched, ... plus any values
the archive adds). --source, --repository and --citation must reference
existing entities; a --citation is also added to the log's own citations list.

Only the file that holds the log is rewritten; the rest of the archive is not
touched. The log ID is printed to stdout.`,
	Example: `  # Record a negative search
  glx add search --log rl-death --source source-findagrave \
    --query "Lewis Little d. 1816-1836 Illinois" --result not_found \
    --date 2026-09-17 --archive ./archive

  # Record a hit with its citation
  glx add search --log rl-death --repository repository-familysearch \
    --collection "Illinois, Probate Records, 1819-1988" --result found \
    --citation citation-probate-lewis-little --archive ./archive

  # Plan a search for later
  glx add search --log rl-death --collection "Rowan County, NC, Wills" \
    --result not_searched --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddSearch,
}

func runAddSearch(_ *cobra.Command, _ []string) error {
	return addSearch(addStreams(), &addSearchOpts)
}

// =============================================================================
// add study
// =============================================================================

var addStudyOpts addStudyOptions

var addStudyCmd = &cobra.Command{
	Use:   "study",
	Short: "Create a study entity",
	Long: `Create a Study entity in the archive.

A study declares the scope of a research project: a one-place or one-name
study, a family reconstruction, a brick wall. --title is required. --type and
--status are validated against the study_types and study_statuses
vocabularies. --place and --source are repeatable and must reference existing
entities; repeated IDs are included once, in the order first supplied.
--date-range is a GLX date string, usually "FROM YYYY TO YYYY".`,
	Example: `  # Brick-wall study
  glx add study --id study-lewis-little-parentage --type brick_wall --status active \
    --title "Parents of Lewis Little" --place place-rowan-nc \
    --date-range "FROM 1750 TO 1822" --archive ./archive

  # One-place study with sources in scope
  glx add study --title "Pohl-Göns One Place Study" --type one_place_study \
    --place place-pohl-goens --source source-pohl-goens-registers \
    --date-range "FROM 1610 TO 1875" --archive ./archive`,
	Args: cobra.NoArgs,
	RunE: runAddStudy,
}

func runAddStudy(_ *cobra.Command, _ []string) error {
	return addStudy(addStreams(), &addStudyOpts)
}

// =============================================================================
// flag registration
// =============================================================================

func init() {
	addCmd.AddCommand(addPersonCmd)
	addCmd.AddCommand(addPlaceCmd)
	addCmd.AddCommand(addEventCmd)
	addCmd.AddCommand(addRepoCmd)
	addCmd.AddCommand(addSourceCmd)
	addCmd.AddCommand(addCitationCmd)
	addCmd.AddCommand(addRelCmd)
	addCmd.AddCommand(addAssertionCmd)
	addCmd.AddCommand(addResearchLogCmd)
	addCmd.AddCommand(addSearchCmd)
	addCmd.AddCommand(addStudyCmd)

	// person
	addCommonFlags(addPersonCmd, &addPersonOpts.addCommonOptions)
	addPersonCmd.Flags().StringVar(&addPersonOpts.Given, "given", "", "Given name(s)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Surname, "surname", "", "Surname")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Prefix, "prefix", "", "Name prefix (e.g. Dr., Rev.)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.SurnamePrefix, "surname-prefix", "", "Surname prefix (e.g. de, von)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Suffix, "suffix", "", "Name suffix (e.g. Jr., III)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Nickname, "nickname", "", "Nickname")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Sex, "sex", "", "Recorded sex (vocabulary key in sex_types)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Gender, "gender", "", "Self-identified gender (vocabulary key in gender_types)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Occupation, "occupation", "", "Occupation (free text)")
	addPersonCmd.Flags().StringVar(&addPersonOpts.Residence, "residence", "", "Place ID of residence (must reference an existing place; person_properties.residence is reference_type:places)")
	addPersonCmd.Flags().StringArrayVar(&addPersonOpts.ExternalIDs, "external-id", nil, "External ID in the form type:value, e.g. wikitree:Little-20642 (repeatable)")

	// place
	addCommonFlags(addPlaceCmd, &addPlaceOpts.addCommonOptions)
	addPlaceCmd.Flags().StringVar(&addPlaceOpts.Name, "name", "", "Place name (required unless --id is given)")
	addPlaceCmd.Flags().StringVar(&addPlaceOpts.Type, "type", "", "Place type (vocabulary key in place_types)")
	addPlaceCmd.Flags().StringVar(&addPlaceOpts.Parent, "parent", "", "Parent place ID")
	addPlaceCmd.Flags().Float64Var(&addPlaceLat, "lat", 0, "Latitude (decimal degrees)")
	addPlaceCmd.Flags().Float64Var(&addPlaceLng, "lng", 0, "Longitude (decimal degrees)")
	addPlaceCmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		addPlaceLatSet = cmd.Flags().Changed("lat")
		addPlaceLngSet = cmd.Flags().Changed("lng")

		return nil
	}

	// event
	addCommonFlags(addEventCmd, &addEventOpts.addCommonOptions)
	addEventCmd.Flags().StringVar(&addEventOpts.Type, "type", "", "Event type (required, vocabulary key in event_types)")
	addEventCmd.Flags().StringVar(&addEventOpts.Date, "date", "", "Event date (GLX date string)")
	addEventCmd.Flags().StringVar(&addEventOpts.Place, "place", "", "Place ID for the event")
	addEventCmd.Flags().StringVar(&addEventOpts.Title, "title", "", "Optional human-readable title")
	addEventCmd.Flags().StringVar(&addEventOpts.Principal, "principal", "", "Person ID of the principal subject (shorthand for --participant PERSON-ID:principal)")
	addEventCmd.Flags().StringArrayVar(&addEventOpts.Participants, "participant", nil, "Additional participant in the form person-id:role (repeatable)")
	addEventCmd.Flags().StringArrayVar(&addEventOpts.Properties, "property", nil, "Event property in the form key=value; key from event_properties (repeatable)")

	// repository
	addCommonFlags(addRepoCmd, &addRepoOpts.addCommonOptions)
	addRepoCmd.Flags().StringVar(&addRepoOpts.Name, "name", "", "Repository name (required unless --id is given)")
	addRepoCmd.Flags().StringVar(&addRepoOpts.Type, "type", "", "Repository type (vocabulary key in repository_types)")
	addRepoCmd.Flags().StringVar(&addRepoOpts.Address, "address", "", "Street address")
	addRepoCmd.Flags().StringVar(&addRepoOpts.City, "city", "", "City")
	addRepoCmd.Flags().StringVar(&addRepoOpts.State, "state", "", "State or province")
	addRepoCmd.Flags().StringVar(&addRepoOpts.PostalCode, "postal-code", "", "Postal code")
	addRepoCmd.Flags().StringVar(&addRepoOpts.Country, "country", "", "Country")
	addRepoCmd.Flags().StringVar(&addRepoOpts.Website, "website", "", "Website URL")

	// source
	addCommonFlags(addSourceCmd, &addSourceOpts.addCommonOptions)
	addSourceCmd.Flags().StringVar(&addSourceOpts.Title, "title", "", "Source title (required unless --id is given)")
	addSourceCmd.Flags().StringVar(&addSourceOpts.Type, "type", "", "Source type (vocabulary key in source_types)")
	addSourceCmd.Flags().StringVar(&addSourceOpts.Repository, "repository", "", "Repository ID")
	addSourceCmd.Flags().StringArrayVar(&addSourceOpts.Authors, "author", nil, "Author (repeatable)")
	addSourceCmd.Flags().StringVar(&addSourceOpts.Date, "date", "", "Publication or compilation date")
	addSourceCmd.Flags().StringVar(&addSourceOpts.Description, "description", "", "Short description")
	addSourceCmd.Flags().StringVar(&addSourceOpts.Language, "language", "", "Source language (e.g. en, de, la)")
	addSourceCmd.Flags().StringVar(&addSourceOpts.URL, "url", "", "Web address of the source collection or database")
	addSourceCmd.Flags().StringVar(&addSourceOpts.PublicationInfo, "publication-info", "", "Publisher, place and date of publication, edition")
	addSourceCmd.Flags().StringVar(&addSourceOpts.CallNumber, "call-number", "", "Repository call number or shelf mark")
	addSourceCmd.Flags().StringVar(&addSourceOpts.SourceNature, "source-nature", "", "Source nature (vocabulary key in source_natures, e.g. original, derivative)")
	addSourceCmd.Flags().StringVar(&addSourceOpts.InformationType, "information-type", "", "Information type (vocabulary key in information_types, e.g. primary, secondary)")

	// citation
	addCommonFlags(addCitationCmd, &addCitationOpts.addCommonOptions)
	addCitationCmd.Flags().StringVar(&addCitationOpts.Source, "source", "", "Source ID (required)")
	addCitationCmd.Flags().StringVar(&addCitationOpts.Repository, "repository", "", "Repository ID (optional override)")
	addCitationCmd.Flags().StringVar(&addCitationOpts.URL, "url", "", "Canonical URL")
	addCitationCmd.Flags().StringVar(&addCitationOpts.Accessed, "accessed", "", "Date the source was accessed (YYYY-MM-DD)")
	addCitationCmd.Flags().StringVar(&addCitationOpts.Locator, "locator", "", "Locator (page number, entry, etc.)")
	addCitationCmd.Flags().StringVar(&addCitationOpts.TextFromSource, "text-from-source", "", "Verbatim transcription from the source")
	addCitationCmd.Flags().StringVar(&addCitationOpts.SourceDate, "source-date", "", "Date carried by the source itself")
	addCitationCmd.Flags().StringArrayVar(&addCitationOpts.ExternalIDs, "external-id", nil, "External ID in the form type:value (repeatable)")
	_ = addCitationCmd.MarkFlagRequired("source")

	// relationship
	addCommonFlags(addRelCmd, &addRelOpts.addCommonOptions)
	addRelCmd.Flags().StringVar(&addRelOpts.Type, "type", "", "Relationship type (required, vocabulary key in relationship_types)")
	addRelCmd.Flags().StringArrayVar(&addRelOpts.Parents, "parent", nil, "Parent person ID (repeatable; adds participant with role parent)")
	addRelCmd.Flags().StringArrayVar(&addRelOpts.Children, "child", nil, "Child person ID (repeatable; adds participant with role child)")
	addRelCmd.Flags().StringArrayVar(&addRelOpts.Spouses, "spouse", nil, "Spouse person ID (repeatable; adds participant with role spouse)")
	addRelCmd.Flags().StringArrayVar(&addRelOpts.Participants, "participant", nil, "Participant in the form person-id:role (repeatable)")
	addRelCmd.Flags().StringVar(&addRelOpts.StartEvent, "start-event", "", "Event ID marking the relationship's start")
	addRelCmd.Flags().StringVar(&addRelOpts.EndEvent, "end-event", "", "Event ID marking the relationship's end")

	// assertion
	addCommonFlags(addAssertionCmd, &addAssertionOpts.addCommonOptions)
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.SubjectPerson, "subject-person", "", "Person ID this assertion is about")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.SubjectEvent, "subject-event", "", "Event ID this assertion is about")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.SubjectRelationship, "subject-relationship", "", "Relationship ID this assertion is about")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.SubjectPlace, "subject-place", "", "Place ID this assertion is about")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.Property, "property", "", "Property name (e.g. date, place, name)")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.Value, "value", "", "Property value")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.Participant, "participant", "", "Participant assertion in the form person-id:role (mutually exclusive with --property/--value)")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.AssertionDate, "date", "", "Date the property value applies to (temporal properties)")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.Confidence, "confidence", "", "Confidence level (vocabulary key in confidence_levels)")
	addAssertionCmd.Flags().StringVar(&addAssertionOpts.Status, "status", "", "Assertion status (e.g. accepted, disputed)")
	addAssertionCmd.Flags().StringArrayVar(&addAssertionOpts.Sources, "source", nil, "Source ID supporting the assertion (repeatable)")
	addAssertionCmd.Flags().StringArrayVar(&addAssertionOpts.Citations, "citation", nil, "Citation ID supporting the assertion (repeatable)")
	addAssertionCmd.Flags().StringArrayVar(&addAssertionOpts.Media, "media", nil, "Media ID supporting the assertion (repeatable)")

	// research-log
	addCommonFlags(addResearchLogCmd, &addResearchLogOpts.addCommonOptions)
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.Title, "title", "", "Human-readable title")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.SubjectPerson, "subject-person", "", "Person ID the log investigates")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.SubjectEvent, "subject-event", "", "Event ID the log investigates")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.SubjectRelationship, "subject-relationship", "", "Relationship ID the log investigates")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.SubjectPlace, "subject-place", "", "Place ID the log investigates")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.Objective, "objective", "", "Research question or goal")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.Status, "status", "", "Status (vocabulary key in research_log_status_types)")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.Date, "date", "", "Date the research was performed (GLX date string)")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.Researcher, "researcher", "", "Name or identifier of the researcher")
	addResearchLogCmd.Flags().StringVar(&addResearchLogOpts.Conclusions, "conclusions", "", "Summary of findings")
	addResearchLogCmd.Flags().StringArrayVar(&addResearchLogOpts.Citations, "citation", nil, "Citation ID produced by this log (repeatable)")

	// search
	addSearchCmd.Flags().StringVarP(&addSearchOpts.ArchivePath, "archive", "a", ".", "Archive path (directory)")
	addSearchCmd.Flags().BoolVar(&addSearchOpts.SkipValidate, "skip-validate", false, "Skip whole-archive validation after adding (vocab and reference checks still run)")
	addSearchCmd.Flags().BoolVar(&addSearchOpts.DryRun, "dry-run", false, "Print what would be changed without writing files")
	addSearchCmd.Flags().StringArrayVar(&addSearchOpts.Notes, "note", nil, "Free-text note about this search (repeatable)")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Log, "log", "", "Research log ID to append to (required)")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Source, "source", "", "Source ID searched")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Repository, "repository", "", "Repository ID searched")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Collection, "collection", "", "Collection name, when there is no Source entity for it")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Query, "query", "", "Search terms used")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Result, "result", "", "Outcome (vocabulary key in search_result_types)")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Citation, "citation", "", "Citation ID produced by the search")
	addSearchCmd.Flags().StringVar(&addSearchOpts.Date, "date", "", "Date the search was performed (GLX date string)")
	_ = addSearchCmd.MarkFlagRequired("log")

	// study
	addCommonFlags(addStudyCmd, &addStudyOpts.addCommonOptions)
	addStudyCmd.Flags().StringVar(&addStudyOpts.Title, "title", "", "Study title (required)")
	addStudyCmd.Flags().StringVar(&addStudyOpts.Type, "type", "", "Study type (vocabulary key in study_types)")
	addStudyCmd.Flags().StringVar(&addStudyOpts.Status, "status", "", "Study status (vocabulary key in study_statuses)")
	addStudyCmd.Flags().StringVar(&addStudyOpts.DateRange, "date-range", "", `Temporal scope as a GLX date string, e.g. "FROM 1750 TO 1822"`)
	addStudyCmd.Flags().StringArrayVar(&addStudyOpts.Places, "place", nil, "Place ID in scope (repeatable)")
	addStudyCmd.Flags().StringArrayVar(&addStudyOpts.Sources, "source", nil, "Source ID in scope (repeatable)")
	_ = addStudyCmd.MarkFlagRequired("title")
}
