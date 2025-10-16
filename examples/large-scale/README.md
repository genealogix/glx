# Large Scale Example

A GENEALOGIX archive designed for performance testing with
10,000+ persons and complex relationship networks.

## Structure

```
large-scale/
├── .oracynth/
│   ├── config.glx
│   └── schema-version.glx
├── persons/
│   ├── person-00001.glx
│   ├── person-00002.glx
│   ├── ...
│   └── person-10000.glx
├── relationships/
│   ├── rel-marriage-00001.glx
│   ├── rel-parent-00001.glx
│   ├── ...
│   └── rel-sibling-05000.glx
├── sources/
│   ├── source-census-1900.glx
│   ├── source-census-1910.glx
│   ├── ...
│   └── source-vital-records-1000.glx
├── media/
│   ├── media-census-1900.glx
│   ├── media-census-1910.glx
│   ├── ...
│   └── media-photos-5000.glx
└── README.md
```

## Performance Testing Overview

This example is designed to test GENEALOGIX implementations with:

- **10,000+ person records** across multiple generations
- **50,000+ relationship records** including marriages, parent-child, and sibling relationships
- **1,000+ source records** representing census data, vital records, and other documents
- **5,000+ media files** including photos, documents, and audio recordings
- **Complex relationship networks** with multiple generations and extended families

## Data Generation Strategy

### Person Records
- **Generation 1**: 2,000 persons (born 1800-1850)
- **Generation 2**: 4,000 persons (born 1850-1900)
- **Generation 3**: 3,000 persons (born 1900-1950)
- **Generation 4**: 1,000 persons (born 1950-2000)

### Relationship Types
- **Marriages**: ~5,000 marriage relationships
- **Parent-Child**: ~15,000 parent-child relationships
- **Siblings**: ~10,000 sibling relationships
- **Extended Family**: ~20,000 extended family relationships

### Source Documents
- **Census Records**: 10 census years (1850-1950)
- **Vital Records**: Birth, marriage, and death certificates
- **Newspaper Articles**: Obituaries and announcements
- **Military Records**: Service records and draft cards

### Media Files
- **Photographs**: Family photos across generations
- **Documents**: Scanned certificates and records
- **Audio**: Oral history interviews
- **Video**: Family gatherings and ceremonies

## Performance Benchmarks

### File System Performance
- **Directory traversal**: Time to scan all .glx files
- **File reading**: Time to load and parse YAML files
- **Memory usage**: RAM consumption for large datasets

### Validation Performance
- **Schema validation**: Time to validate all files
- **Relationship validation**: Time to verify relationship integrity
- **Cross-reference checking**: Time to validate all references

### Query Performance
- **Person lookup**: Time to find specific persons
- **Relationship traversal**: Time to follow relationship chains
- **Source correlation**: Time to find all sources for a person

## Implementation Notes

### File Naming Convention
- **Persons**: `person-XXXXX.glx` (5-digit zero-padded)
- **Relationships**: `rel-{type}-XXXXX.glx`
- **Sources**: `source-{type}-XXXXX.glx`
- **Media**: `media-{type}-XXXXX.glx`

### Relationship Integrity
- All person IDs referenced in relationships must exist
- All source IDs referenced in assertions must exist
- All media IDs referenced in sources must exist
- Circular references are avoided in relationship chains

### Data Consistency
- Birth dates are consistent with parent ages
- Marriage dates are consistent with spouse ages
- Death dates are consistent with life events
- Geographic locations are historically accurate

## Validation

```bash
# Validate all files (may take several minutes)
glx validate

# Check specific performance metrics
time glx validate persons/
time glx validate relationships/
time glx validate sources/
time glx validate media/
```

## Performance Testing Scripts

### Basic Validation Test
```bash
#!/bin/bash
echo "Testing large-scale validation performance..."
time glx validate
echo "Validation complete"
```

### Memory Usage Test
```bash
#!/bin/bash
echo "Testing memory usage..."
/usr/bin/time -v glx validate 2>&1 | grep "Maximum resident set size"
```

### File Count Test
```bash
#!/bin/bash
echo "Counting files..."
find . -name "*.glx" | wc -l
echo "Files found"
```

## What This Demonstrates

- **Scalability**: How GENEALOGIX performs with large datasets
- **Memory efficiency**: RAM usage patterns for big data
- **Validation speed**: Time to validate thousands of files
- **Relationship integrity**: Maintaining data consistency at scale
- **File organization**: Efficient directory structures for large archives

## Implementation Requirements

To handle large-scale archives, implementations should:

1. **Stream processing**: Process files without loading everything into memory
2. **Indexing**: Create indexes for fast person and relationship lookup
3. **Caching**: Cache frequently accessed data
4. **Parallel processing**: Use multiple threads for validation
5. **Progress reporting**: Show progress for long-running operations

## Next Steps

Expand the large-scale example:
- Add more relationship types (adoption, step-family, etc.)
- Include international families with multiple countries
- Add DNA evidence and genetic genealogy data
- Create performance regression tests
- Document optimization strategies
