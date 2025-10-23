# Complete Family Example

This example demonstrates the full GENEALOGIX specification with all 9 entity types:

- **Persons**: Individual family members
- **Relationships**: Family connections (marriage, parent-child)
- **Events**: Births, marriages, occupations, and other facts
- **Places**: Geographic locations with hierarchies
- **Sources**: Bibliographic references
- **Citations**: Specific evidence references with quality ratings
- **Repositories**: Archives and institutions holding sources
- **Assertions**: Evidence-based conclusions
- **Media**: Photos and documents

## Structure

```
persons/              - Individual records (John, Sarah, Jane)
relationships/        - Relationships (marriage, parent-child)
events/              - Life events (births, marriages, occupations)
places/              - Geographic locations (England, Yorkshire, Leeds)
sources/             - Bibliographic sources (parish registers, census)
citations/           - Specific evidence references
repositories/        - Archives and institutions
assertions/          - Conclusions supported by evidence
media/               - Photographs and documents
```

## Key Features Demonstrated

### Multi-Generation Family
- John Smith & Mary Brown: marriage, children
- Their children: births, marriages, occupations
- Grandchildren: births and early life events

### Evidence Documentation
- Parish register citations (birth records)
- Census records with page numbers
- Quality ratings (GEDCOM QUAY 0-3)
- Structured locators (film numbers, URLs)

### Geographic Hierarchy
- Country: England
- County: Yorkshire  
- Cities: Leeds, Bradford
- Places referenced in events

### Complete Evidence Chain
- Source → Citation → Assertion
- Each assertion backed by quality-rated citations
- Confidence scoring on conclusions

## Validation

Validate all files:
```bash
glx validate
```

Validate specific entity type:
```bash
glx validate events/
glx validate places/
glx validate citations/
```

## Data Model

### Person → Event → Assertion Chain
1. Person "John Smith" (person-john001)
2. Birth event (event-birth-john) at place (place-leeds)
3. Assertion about birth date backed by citation
4. Citation references parish register (source-parish-leeds)
5. Source held at repository (repository-leeds-lib)

### Multi-Person Event
Marriage of John & Mary:
1. Relationship record connects them (rel-marriage-john-mary)
2. Marriage event (event-marriage-001) with both as participants
3. Assertions about marriage date & place
4. Citations to marriage register

## Files

### Persons (3 records)
- person-john-smith.glx - John Smith (1850-1920)
- person-mary-brown.glx - Mary Brown (1852-1930)
- person-jane-smith.glx - Jane Smith (1875-1955), daughter

### Relationships (2 records)
- rel-john-mary-marriage.glx - John & Mary married 1875
- rel-john-jane-parent.glx - John is Jane's father

### Events (6 records)
- event-john-birth.glx - John born Jan 15 1850
- event-mary-birth.glx - Mary born Mar 20 1852
- event-john-mary-marriage.glx - Married May 10 1875
- event-john-occupation.glx - Blacksmith 1870-1920
- event-jane-birth.glx - Jane born Sep 5 1875
- event-jane-occupation.glx - Dressmaker 1895-1920

### Places (3 records)
- place-england.glx - England (country)
- place-yorkshire.glx - Yorkshire (county, parent: England)
- place-leeds.glx - Leeds (city, parent: Yorkshire)

### Sources (2 records)
- source-parish-leeds.glx - St Paul's Church Parish Registers
- source-census-1851.glx - 1851 Census Records

### Citations (4 records)
- citation-john-birth.glx - References parish register, quality 3 (primary)
- citation-mary-birth.glx - References parish register, quality 3
- citation-marriage.glx - References parish register, quality 3
- citation-1851-census.glx - References census, quality 2 (secondary)

### Repositories (2 records)
- repository-leeds-library.glx - Leeds Library Local Studies
- repository-tna.glx - The National Archives

### Assertions (6 records)
- assertion-john-birth-date.glx - John born 1850
- assertion-john-birth-place.glx - John born in Leeds
- assertion-john-occupation.glx - John was a blacksmith
- assertion-jane-birth-date.glx - Jane born 1875
- assertion-marriage-date.glx - Marriage date May 1875
- assertion-marriage-place.glx - Marriage in Leeds

## Testing

Run validation:
```bash
cd examples/complete-family
glx validate
```

Should report all files valid with no errors.

## Next Steps

1. Study individual files to understand entity structure
2. Review how entities reference each other
3. Check how evidence chains work (Source → Citation → Assertion)
4. Note hierarchical relationships (Places, Repositories)

For detailed format specifications, see the main GENEALOGIX documentation.




