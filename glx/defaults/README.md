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
- rel-XXXXXXXX: Relationship records
- event-XXXXXXXX: Event/Fact records
- place-XXXXXXXX: Place records
- assertion-XXXXXXXX: Assertion records
- source-XXXXXXXX: Source records
- citation-XXXXXXXX: Citation records
- repository-XXXXXXXX: Repository records
- media-XXXXXXXX: Media records

## Documentation

See the [GENEALOGIX specification](https://github.com/genealogix/glx) for detailed format information.

