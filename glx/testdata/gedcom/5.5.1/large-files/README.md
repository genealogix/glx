# Large File Performance Testing (GEDCOM 5.5.1)

## Description
Collection of large GEDCOM 5.5.1 files for performance testing, stress testing, and validating parser handling of substantial genealogy databases. These files test the parser's ability to handle multi-megabyte datasets efficiently and without memory issues.

## Test Case Type
- **Performance testing**: Large file processing speed
- **Memory testing**: Memory usage with large datasets
- **Stress testing**: Parser robustness with extensive data
- **Scalability testing**: Handling of thousands of individuals
- **Real-world data**: Actual genealogy databases

## File Information
- **GEDCOM Version**: 5.5 / 5.5.1
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: 2.5 MB to 10 MB
- **Encoding**: UTF-8, Unknown-8bit
- **Source Software**: Family Tree Maker 20.0, MyHeritage

## Provenance
- **Source**: findmypast/gedcom-samples
- **URL**: https://github.com/findmypast/gedcom-samples
- **Availability**: Open source test files
- **Purpose**: Performance and stress testing
- **Collection**: Real-world genealogy databases

## License & Usage
- **License**: Open source test files
- **Attribution**: findmypast
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to repository license

## Test Files

### habsburg.ged
- **Family**: Habsburg royal family genealogy
- **Size**: 10 MB (largest test file)
- **Encoding**: Unknown-8bit
- **Software**: Family Tree Maker 20.0
- **Description**: Extensive Habsburg dynasty genealogy
- **Individuals**: Thousands of royal family members
- **Relationships**: Complex royal intermarriage networks
- **Coverage**: Centuries of European royal history
- **Purpose**: Maximum stress testing
- **Performance**: Tests parser with very large datasets

### queen.ged
- **Family**: British royal genealogy
- **Size**: 2.5 MB
- **Encoding**: UTF-8 (with BOM)
- **Software**: MyHeritage 8.0.0.8367
- **Export Date**: 17 June 2017
- **Description**: Large royal genealogy database with HTML-formatted notes
- **Individuals**: Extensive royal lineages
- **Purpose**: Large file performance testing
- **Known Issues**: Contains malformed HTML in NOTE fields (line 15903 missing CONT prefix)
- **Performance**: Tests medium-large file handling and error recovery

## Testing Coverage
- **Large file parsing**: Multi-megabyte GEDCOM files
- **Memory efficiency**: RAM usage with extensive data
- **Processing speed**: Parse time for large files
- **Entity handling**: Thousands of individuals and families
- **Relationship complexity**: Complex family networks
- **Encoding handling**: UTF-8 and legacy encodings
- **Buffer management**: Large file buffering strategies
- **Progress reporting**: Progress indication for long operations
- **Error handling**: Error recovery in large files
- **Resource cleanup**: Memory cleanup after processing

## Performance Benchmarks

### Expected Performance Characteristics
- **Habsburg.ged (10 MB)**:
  - Should parse in under 30 seconds on modern hardware
  - Memory usage should remain under 500 MB
  - All individuals and relationships should be extracted
  - No memory leaks or crashes

- **Queen.ged (2.5 MB)**:
  - Should parse in under 10 seconds on modern hardware
  - Memory usage should remain under 200 MB
  - Complete data extraction
  - Stable performance

## Notes
- **Essential for performance validation**: Tests real-world file sizes
- **Habsburg.ged is the largest test**: 10 MB stress test
- **Memory profiling recommended**: Monitor memory during parsing
- **Timeout testing**: Ensure parser completes in reasonable time
- **Progress indicators**: Good candidate for progress reporting
- **Resource limits**: Tests parser resource management
- **Real genealogy data**: Actual family trees, not synthetic
- **Complex relationships**: Tests relationship graph handling
- **UTF-8 BOM handling**: Queen.ged tests BOM detection
- **Legacy encoding**: Habsburg.ged tests older encoding handling
- **Regression testing**: Detect performance degradation
- **Optimization target**: Benchmark for optimization efforts

## Usage Recommendations
1. Use for performance regression testing
2. Profile memory usage during parsing
3. Measure parsing speed improvements
4. Test progress reporting mechanisms
5. Validate handling of large entity counts
6. Stress test relationship graph building
7. Test resource cleanup and garbage collection
8. Benchmark against other GEDCOM parsers
9. Validate timeout and cancellation handling
10. Test streaming/chunked parsing strategies
