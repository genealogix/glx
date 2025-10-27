package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "check-schemas":
		if err := runCheckSchemas(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "init":
		singleFile := false
		if len(os.Args) > 2 && (os.Args[2] == "--single-file" || os.Args[2] == "-s") {
			singleFile = true
		}
		if err := runInit(singleFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "validate":
		if err := runValidate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("genealogix CLI")
	fmt.Println("Usage:")
	fmt.Println("  glx init [--single-file]  Initialize a new genealogix repository")
	fmt.Println("                            (default: multi-file with folders)")
	fmt.Println("  glx validate [paths]      Validate .glx files and cross-references")
	fmt.Println("  glx check-schemas         Validate schema files for required metadata")
}

func runInit(singleFile bool) error {
	// Check if we're in the GENEALOGIX spec repository (not a user archive)
	if isSpecRepository() {
		return fmt.Errorf("cannot run 'glx init' in the GENEALOGIX specification repository. Create a new directory for your family archive first")
	}

	if singleFile {
		// Create single-file archive template
		template := `# GENEALOGIX Family Archive
# Single-file format

persons: {}
relationships: {}
events: {}
places: {}
sources: {}
citations: {}
repositories: {}
assertions: {}
media: {}
`
		if err := os.WriteFile("archive.glx", []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to create archive.glx: %v", err)
		}

		fmt.Println("Initialized single-file GENEALOGIX archive: archive.glx")
		fmt.Println("Add entities under the appropriate type keys (persons, sources, etc.)")
		fmt.Println("Entity IDs are map keys - don't include 'id' field in entities")
		return nil
	}

	// Create directory structure for a genealogix repository
	dirs := []string{
		"persons",
		"relationships",
		"events",
		"places",
		"sources",
		"citations",
		"repositories",
		"assertions",
		"media",
		"vocabularies",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create standard vocabulary files
	if err := createStandardVocabularies(); err != nil {
		return err
	}

	// Create .gitignore file
	gitignore := `# GENEALOGIX Repository
# Ignore temporary files and build artifacts

*.tmp
*.bak
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db
`

	if err := os.WriteFile(".gitignore", []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %v", err)
	}

	// Create README.md for the repository
	readme := `# GENEALOGIX Family Archive

This is a genealogical archive using the GENEALOGIX format.

## Structure

### Core Data
- persons/ - Individual person records
- relationships/ - Family relationships and connections
- events/ - Life events (births, marriages, deaths, occupations, residences, etc.)
- places/ - Geographic locations with hierarchies

### Evidence & Sources
- sources/ - Bibliographic sources and publications
- citations/ - Specific references within sources
- repositories/ - Archives, libraries, and institutions holding sources
- assertions/ - Evidence-based conclusions and claims

### Media
- media/ - Photos, documents, and other media files

## Getting Started

Use glx commands to work with this archive:

` + "```bash" + `
glx validate              # Validate all .glx files
glx validate persons/     # Validate specific directory
` + "```" + `

## File Format

All genealogical data is stored in YAML files with the .glx extension.
Each file represents a specific entity (person, event, place, citation, etc.).

### Standard ID Prefixes
- person-XXXXXXXX: Person records
- rel-XXXXXXXX: Relationship records
- event-XXXXXXXX: Event/Fact records
- place-XXXXXXXX: Place records
- assertion-XXXXXXXX: Assertion records
- source-XXXXXXXX: Source records
- citation-XXXXXXXX: Citation records
- repository-XXXXXXXX: Repository records
- media-XXXXXXXX: Media records

## Documentation

See the [GENEALOGIX specification](https://github.com/genealogix/spec) for detailed format information.
`

	if err := os.WriteFile("README.md", []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %v", err)
	}

	fmt.Println("Initialized multi-file GENEALOGIX repository")
	fmt.Println("Created directories:")
	fmt.Println("  Core: persons/, relationships/, events/, places/")
	fmt.Println("  Evidence: sources/, citations/, repositories/, assertions/")
	fmt.Println("  Media: media/")
	fmt.Println("Created .gitignore and README.md")
	fmt.Println("")
	fmt.Println("Each .glx file should have entity type keys at the top level.")
	fmt.Println("Entity IDs are map keys - don't include 'id' field in entities.")

	return nil
}

func isSpecRepository() bool {
	// Check if we're in the GENEALOGIX spec repository by looking for key files
	specFiles := []string{"specification/README.md", "specification/schema/v1/person.schema.json", "glx/main.go"}
	for _, file := range specFiles {
		if _, err := os.Stat(file); err != nil {
			return false
		}
	}
	return true
}

func createStandardVocabularies() error {
	// Define standard vocabulary templates
	relationshipTypesTemplate := `# GENEALOGIX Relationship Types Vocabulary
# Standard relationship types for genealogical research

relationship_types:
  marriage:
    label: "Marriage"
    description: "Legal or religious union of two people"
    gedcom: "MARR"
  parent-child:
    label: "Parent-Child"
    description: "Biological, adoptive, or legal parent-child relationship"
    gedcom: "CHIL/FAMC"
  sibling:
    label: "Sibling"
    description: "Brother or sister relationship"
    gedcom: "SIB"
  adoption:
    label: "Adoption"
    description: "Legal adoption relationship"
    gedcom: "ADOP"
  step-parent:
    label: "Step-Parent"
    description: "Step-parent relationship through marriage"
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor relationship"
  guardian:
    label: "Guardian"
    description: "Legal guardian relationship"
  partner:
    label: "Partner"
    description: "Domestic partnership or cohabitation"

# Add custom relationship types below:
# custom-type:
#   label: "Display Name"
#   description: "Description of relationship"
#   custom: true
`

	eventTypesTemplate := `# GENEALOGIX Event Types Vocabulary
# Standard event types for genealogical research

event_types:
  birth:
    label: "Birth"
    description: "Person's birth"
    category: "lifecycle"
    gedcom: "BIRT"
  death:
    label: "Death"
    description: "Person's death"
    category: "lifecycle"
    gedcom: "DEAT"
  marriage:
    label: "Marriage"
    description: "Union of two people"
    category: "lifecycle"
    gedcom: "MARR"
  divorce:
    label: "Divorce"
    description: "Dissolution of marriage"
    category: "lifecycle"
    gedcom: "DIV"
  engagement:
    label: "Engagement"
    description: "Betrothal or engagement"
    category: "lifecycle"
    gedcom: "ENGA"
  adoption:
    label: "Adoption"
    description: "Adoption event"
    category: "lifecycle"
    gedcom: "ADOP"
  baptism:
    label: "Baptism"
    description: "Religious baptism ceremony"
    category: "religious"
    gedcom: "BAPM"
  confirmation:
    label: "Confirmation"
    description: "Religious confirmation"
    category: "religious"
    gedcom: "CONF"
  bar_mitzvah:
    label: "Bar Mitzvah"
    description: "Jewish coming of age ceremony"
    category: "religious"
    gedcom: "BARM"
  bat_mitzvah:
    label: "Bat Mitzvah"
    description: "Jewish coming of age ceremony"
    category: "religious"
    gedcom: "BATM"
  burial:
    label: "Burial"
    description: "Interment of remains"
    category: "lifecycle"
    gedcom: "BURI"
  cremation:
    label: "Cremation"
    description: "Cremation of remains"
    category: "lifecycle"
    gedcom: "CREM"
  residence:
    label: "Residence"
    description: "Place of residence"
    category: "attribute"
    gedcom: "RESI"
  occupation:
    label: "Occupation"
    description: "Employment or trade"
    category: "attribute"
    gedcom: "OCCU"
  title:
    label: "Title"
    description: "Nobility or honorific title"
    category: "attribute"
    gedcom: "TITL"
  nationality:
    label: "Nationality"
    description: "National citizenship"
    category: "attribute"
    gedcom: "NATI"
  religion:
    label: "Religion"
    description: "Religious affiliation"
    category: "attribute"
    gedcom: "RELI"
  education:
    label: "Education"
    description: "Educational achievement"
    category: "attribute"
    gedcom: "EDUC"

# Add custom event types below:
# custom-event:
#   label: "Display Name"
#   description: "Description of event"
#   category: "custom"
#   custom: true
`

	placeTypesTemplate := `# GENEALOGIX Place Types Vocabulary
# Standard place types for geographic hierarchy

place_types:
  country:
    label: "Country"
    description: "Nation state or country"
    category: "administrative"
  state:
    label: "State"
    description: "State, province, or major administrative division"
    category: "administrative"
  county:
    label: "County"
    description: "County or shire"
    category: "administrative"
  city:
    label: "City"
    description: "Major urban area or city"
    category: "geographic"
  town:
    label: "Town"
    description: "Smaller urban area or town"
    category: "geographic"
  village:
    label: "Village"
    description: "Small rural community"
    category: "geographic"
  parish:
    label: "Parish"
    description: "Church parish or ecclesiastical division"
    category: "religious"
  registration_district:
    label: "Registration District"
    description: "Civil registration area"
    category: "administrative"
  address:
    label: "Address"
    description: "Specific street address"
    category: "geographic"
  building:
    label: "Building"
    description: "Specific building or structure"
    category: "geographic"
  street:
    label: "Street"
    description: "Street or road"
    category: "geographic"

# Add custom place types below:
# custom-place:
#   label: "Display Name"
#   description: "Description of place type"
#   category: "custom"
#   custom: true
`

	repositoryTypesTemplate := `# GENEALOGIX Repository Types Vocabulary
# Standard repository types for holding genealogical sources

repository_types:
  archive:
    label: "Archive"
    description: "Government or historical archive"
  library:
    label: "Library"
    description: "Public, university, or specialty library"
  museum:
    label: "Museum"
    description: "Museum with genealogical collections"
  registry:
    label: "Registry"
    description: "Civil registration office or vital records office"
  database:
    label: "Database"
    description: "Online genealogical database service"
  church:
    label: "Church"
    description: "Church or religious organization archives"
  historical_society:
    label: "Historical Society"
    description: "Local historical society"
  university:
    label: "University"
    description: "University special collections or archives"
  government_agency:
    label: "Government Agency"
    description: "Government record-keeping agency"

# Add custom repository types below:
# custom-repo:
#   label: "Display Name"
#   description: "Description of repository type"
#   custom: true
`

	participantRolesTemplate := `# GENEALOGIX Participant Roles Vocabulary
# Standard roles for event participants

participant_roles:
  principal:
    label: "Principal"
    description: "Primary person involved in the event"
  principal1:
    label: "Principal 1"
    description: "First primary person (e.g., in marriage)"
  principal2:
    label: "Principal 2"
    description: "Second primary person (e.g., in marriage)"
  groom:
    label: "Groom"
    description: "Male spouse in marriage"
  bride:
    label: "Bride"
    description: "Female spouse in marriage"
  witness:
    label: "Witness"
    description: "Witness to the event"
  officiant:
    label: "Officiant"
    description: "Person officiating the event"
  celebrant:
    label: "Celebrant"
    description: "Person celebrating the event"
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor"
  proxy:
    label: "Proxy"
    description: "Person acting as proxy for another"
  informant:
    label: "Informant"
    description: "Person providing information"
  attending:
    label: "Attending"
    description: "Person present at the event"

# Add custom participant roles below:
# custom-role:
#   label: "Display Name"
#   description: "Description of role"
#   custom: true
`

	mediaTypesTemplate := `# GENEALOGIX Media Types Vocabulary
# Standard media types for genealogical materials

media_types:
  photo:
    label: "Photograph"
    description: "Still image photograph"
    mime_type: "image/jpeg"
  document:
    label: "Document"
    description: "Text document or scanned document"
    mime_type: "application/pdf"
  audio:
    label: "Audio"
    description: "Audio recording"
    mime_type: "audio/mpeg"
  video:
    label: "Video"
    description: "Video recording"
    mime_type: "video/mp4"
  certificate:
    label: "Certificate"
    description: "Official certificate document"
    mime_type: "application/pdf"
  letter:
    label: "Letter"
    description: "Personal or official letter"
    mime_type: "application/pdf"
  newspaper:
    label: "Newspaper"
    description: "Newspaper article or clipping"
    mime_type: "image/jpeg"
  census:
    label: "Census"
    description: "Census record"
    mime_type: "image/jpeg"

# Add custom media types below:
# custom-media:
#   label: "Display Name"
#   description: "Description of media type"
#   mime_type: "application/custom"
#   custom: true
`

	confidenceLevelsTemplate := `# GENEALOGIX Confidence Levels Vocabulary
# Standard confidence levels for genealogical conclusions

confidence_levels:
  high:
    label: "High"
    description: "Multiple primary sources agree"
  medium:
    label: "Medium"
    description: "Some conflicting evidence, but preponderance supports"
  low:
    label: "Low"
    description: "Limited evidence, requires more research"
  disputed:
    label: "Disputed"
    description: "Multiple sources conflict, resolution unclear"

# Add custom confidence levels below:
# custom-level:
#   label: "Display Name"
#   description: "Description of confidence level"
#   custom: true
`

	qualityRatingsTemplate := `# GENEALOGIX Quality Ratings Vocabulary
# Standard quality ratings for evidence reliability

quality_ratings:
  0:
    label: "Unreliable"
    description: "Unreliable evidence or estimated data"
    gedcom: "0"
  1:
    label: "Questionable"
    description: "Questionable reliability of evidence"
    gedcom: "1"
  2:
    label: "Secondary"
    description: "Secondary evidence, officially recorded"
    gedcom: "2"
  3:
    label: "Primary"
    description: "Direct and primary evidence"
    gedcom: "3"

# Add custom quality ratings below:
# 4:
#   label: "Custom Quality"
#   description: "Custom quality rating"
#   custom: true
`

	vocabFiles := map[string]string{
		"vocabularies/relationship-types.glx": relationshipTypesTemplate,
		"vocabularies/event-types.glx":        eventTypesTemplate,
		"vocabularies/place-types.glx":        placeTypesTemplate,
		"vocabularies/repository-types.glx":   repositoryTypesTemplate,
		"vocabularies/participant-roles.glx":  participantRolesTemplate,
		"vocabularies/media-types.glx":        mediaTypesTemplate,
		"vocabularies/confidence-levels.glx":  confidenceLevelsTemplate,
		"vocabularies/quality-ratings.glx":    qualityRatingsTemplate,
	}

	for path, template := range vocabFiles {
		if err := os.WriteFile(path, []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %v", path, err)
		}
	}

	fmt.Println("Created standard vocabulary files in vocabularies/")
	return nil
}
