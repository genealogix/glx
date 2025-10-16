# Evidence-Based Example

A GENEALOGIX archive demonstrating proper source citation,
evidence evaluation, and assertion-based research methodology.

## Structure

```
evidence-based/
├── .oracynth/
│   ├── config.glx
│   └── schema-version.glx
├── persons/
│   ├── person-ancestor.glx
│   ├── person-ancestor-wife.glx
│   └── person-descendant.glx
├── relationships/
│   ├── rel-marriage-william.glx
│   └── rel-parent-john.glx
├── sources/
│   ├── source-birth-certificate.glx
│   ├── source-marriage-record.glx
│   └── source-census-1900.glx
├── media/
│   ├── media-birth-cert-scan.glx
│   ├── media-marriage-license.glx
│   └── media-census-1900.glx
├── assertions/
│   ├── assert-birth-john.glx
│   ├── assert-birth-william.glx
│   ├── assert-birth-mary.glx
│   ├── assert-marriage-william.glx
│   ├── assert-marriage-mary.glx
│   ├── assert-death-william.glx
│   └── assert-death-mary.glx
└── README.md
```

## Research Overview

This example demonstrates proper genealogical research methodology:

- **William Smith** (b. 1935, d. 2010) and **Mary Johnson** (b. 1938, d. 2015) married in 1960
- They had a son **John Smith** (b. 1985)
- Each fact is supported by appropriate source documents
- Confidence levels are assigned based on source quality

## Key Concepts Demonstrated

### Source Documentation
- **Primary sources**: Birth certificates, marriage licenses
- **Secondary sources**: Census records, family records
- **Media attachments**: Scanned documents with hash verification
- **Proper citations**: Author, date, repository information

### Evidence Evaluation
- **High confidence**: Official vital records (birth, marriage, death certificates)
- **Medium confidence**: Census records, family knowledge
- **Source correlation**: Multiple sources supporting the same fact

### Assertion-Based Research
- Each genealogical fact is documented as an assertion
- Assertions reference supporting sources
- Confidence levels indicate reliability
- Clear separation between evidence and conclusions

## Files

### sources/source-birth-certificate.glx
```yaml
id: source-birth-cert
version: "1.0"
title: "Birth Certificate - John Smith"
type: "Vital Record"
authors: ["State of California"]
date: "1985-03-15"
citation: "California Department of Health Services, Birth Certificate #85-123456"

media:
  - media/media-birth-cert-scan.glx
```

### assertions/assert-birth-john.glx
```yaml
id: assert-birth-john
version: "1.0"
type: birth
subject: person-descendant
date: "1985-03-15"
place: "Sacramento, California, USA"
confidence: high

sources:
  - sources/source-birth-certificate.glx
```

### media/media-birth-cert-scan.glx
```yaml
id: media-birth-cert-scan
version: "1.0"
uri: "media/birth-certificate-john-smith-1985.pdf"
mime_type: "application/pdf"
hash: "sha256:abc123def456..."
```

## Validation

```bash
glx validate
# ✓ All files valid

glx check-schemas
# ✓ schemas valid
```

## What This Demonstrates

- **Source hierarchy**: Primary vs. secondary sources
- **Evidence evaluation**: Confidence levels and source quality
- **Citation standards**: Proper attribution and repository information
- **Media management**: File attachments with integrity verification
- **Research methodology**: Assertion-based genealogical research
- **Source correlation**: Multiple sources supporting conclusions

## Research Standards

This example follows genealogical best practices:

1. **Source every fact**: No unsupported claims
2. **Evaluate evidence**: Assign confidence levels
3. **Cite properly**: Include all necessary citation elements
4. **Verify integrity**: Use hash verification for media files
5. **Separate evidence from conclusions**: Clear distinction between sources and assertions

## Next Steps

Expand the research:
- Add more census records for different years
- Include newspaper articles and obituaries
- Add DNA evidence and genetic genealogy
- Document research methodology and reasoning
