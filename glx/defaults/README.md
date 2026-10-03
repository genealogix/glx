# GENEALOGIX Family Archive

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

### Research
- research_logs/ - Research logs: what was searched, where, and what was found
- studies/ - Research studies (one-name, one-place, FAN-club) and their scope

## Getting Started

Use glx commands to work with this archive:

```bash
glx validate              # Validate all .glx files
glx validate persons/     # Validate specific directory
```

## File Format

All genealogical data is stored in YAML files with the .glx extension.
Each file represents a specific entity (person, event, place, citation, etc.).

### Standard ID Prefixes
- person-XXXXXXXX: Person records
- relationship-XXXXXXXX: Relationship records
- event-XXXXXXXX: Event/Fact records
- place-XXXXXXXX: Place records
- assertion-XXXXXXXX: Assertion records
- source-XXXXXXXX: Source records
- citation-XXXXXXXX: Citation records
- repository-XXXXXXXX: Repository records
- media-XXXXXXXX: Media records
- research-log-XXXXXXXX: Research log records
- study-XXXXXXXX: Study records

## Documentation

See the [GENEALOGIX specification](https://genealogix.io/) for detailed format information.

