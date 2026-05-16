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

Every subcommand validates supplied values against the archive's vocabularies
and entity references before writing. The created entity ID is echoed on its
own line as the final stdout output so it can be captured with shell
substitution:

    person_id=$(glx add person --given Johann --surname Jungk --archive .)
    glx add event --type christening --principal "$person_id" --archive .`,
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
them.`,
	Example: `  # Mint a person from given + surname
  glx add person --given "Johann Peter" --surname "Jungk" --sex male --archive ./archive

  # Capture the ID into a shell variable
  pid=$(glx add person --given Anna --surname Müller --archive ./archive)
  echo "created $pid"

  # Override the ID explicitly
  glx add person --id person-jungk-johann-peter --given Johann --surname Jungk --archive ./archive`,
	RunE: runAddPerson,
}

func runAddPerson(_ *cobra.Command, _ []string) error {
	return addPerson(SystemIOStreams(), &addPersonOpts)
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
	RunE: runAddPlace,
}

func runAddPlace(_ *cobra.Command, _ []string) error {
	if addPlaceLatSet {
		addPlaceOpts.Latitude = &addPlaceLat
	}
	if addPlaceLngSet {
		addPlaceOpts.Longitude = &addPlaceLng
	}

	return addPlace(SystemIOStreams(), &addPlaceOpts)
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
--participant is repeatable in the form person-id:role.`,
	Example: `  # Christening with a principal and a place
  glx add event --type christening --date 1725-02-25 \
    --place place-enkirch \
    --principal person-johann-peter-jungk \
    --archive ./archive

  # Marriage with multiple participants
  glx add event --type marriage --date 1850 \
    --participant person-john:groom \
    --participant person-jane:bride \
    --archive ./archive`,
	RunE: runAddEvent,
}

func runAddEvent(_ *cobra.Command, _ []string) error {
	return addEvent(SystemIOStreams(), &addEventOpts)
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
	RunE: runAddRepository,
}

func runAddRepository(_ *cobra.Command, _ []string) error {
	return addRepository(SystemIOStreams(), &addRepoOpts)
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
vocabulary. --repository must reference an existing repository.`,
	Example: `  # Indexed database source
  glx add source --title "Deutschland Geburten und Taufen, 1558-1898" \
    --type database --repository repository-familysearch --archive ./archive

  # Book with author(s) and date
  glx add source --title "Genealogy of the Smith Family" \
    --type book --author "Jane Smith" --date 1923 --archive ./archive`,
	RunE: runAddSource,
}

func runAddSource(_ *cobra.Command, _ []string) error {
	return addSource(SystemIOStreams(), &addSourceOpts)
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
	RunE: runAddCitation,
}

func runAddCitation(_ *cobra.Command, _ []string) error {
	return addCitation(SystemIOStreams(), &addCitationOpts)
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
	RunE: runAddRelationship,
}

func runAddRelationship(_ *cobra.Command, _ []string) error {
	return addRelationship(SystemIOStreams(), &addRelOpts)
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
	RunE: runAddAssertion,
}

func runAddAssertion(_ *cobra.Command, _ []string) error {
	return addAssertion(SystemIOStreams(), &addAssertionOpts)
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
}
