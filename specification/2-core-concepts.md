# Core Concepts

This section explains the fundamental principles and architecture that make GENEALOGIX different from traditional genealogy formats.

## Assertion-Aware Data Model

### The Problem with Traditional Models
Traditional genealogy software stores conclusions directly:
```
Person: John Smith
Birth: January 15, 1850
Place: Leeds, Yorkshire
```

This approach loses the critical distinction between **evidence** (what sources say) and **conclusions** (what we believe). If conflicting evidence emerges, there's no clear way to represent uncertainty or evaluate source quality.

### GENEALOGIX Solution: Assertions
GENEALOGIX separates evidence from conclusions using **assertions**:

```yaml
# assertions/assertion-john-birth.glx - Conclusion based on evidence
assertions:
  assertion-john-birth:
    version: "1.0"
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
      - citation-baptism-record
    confidence: high

# citations/citation-birth-certificate.glx - Specific evidence
citations:
  citation-birth-certificate:
    version: "1.0"
    source: source-gro-register
    locator: "Certificate 1850-LEEDS-00145"
    quality: 3
    transcription: "John Smith, born January 15, 1850"
```

### Benefits of This Approach
1. **Multiple Evidence**: One assertion can reference multiple citations
2. **Quality Assessment**: Each citation includes evidence quality (0-3 scale)
3. **Conflicting Evidence**: Multiple assertions can exist for the same fact
4. **Research Transparency**: Clear audit trail from source to conclusion
5. **Confidence Levels**: Assertions can express certainty based on evidence

## Evidence Hierarchy

### Four Dimensions of Evidence Quality
GENEALOGIX evaluates evidence along four dimensions:

#### 1. Primary vs Secondary
- **Primary**: Created at the time of the event
  - Birth certificates, baptism records, marriage licenses
  - Contemporary letters, diaries, newspapers
- **Secondary**: Created later, based on primary sources
  - Census records, compiled databases, published histories

#### 2. Direct vs Indirect
- **Direct**: Explicitly states the fact you're trying to prove
  - Birth certificate showing birth date (direct evidence of birth)
- **Indirect**: Requires inference or additional information
  - Census showing age 25 in 1875 (indirect evidence of 1850 birth)

#### 3. Original vs Derivative
- **Original**: First-hand, eyewitness account
  - Handwritten parish register entry
- **Derivative**: Copy, transcription, or compilation
  - Published transcription of parish register

#### 4. Information vs Evidence
- **Information**: Raw data from a source
- **Evidence**: Information + analysis + correlation with other evidence

### Quality Rating System
GENEALOGIX uses a 0-3 quality scale compatible with GEDCOM QUAY:

| Rating | Description | Example |
|--------|-------------|---------|
| **3** | Direct and primary evidence | Original birth certificate, contemporary baptism record |
| **2** | Secondary evidence, officially recorded | Census record, published vital records index |
| **1** | Questionable reliability | Undocumented oral history, conflicting sources |
| **0** | Unreliable or estimated | Unverified family tradition, unsourced data |

**GEDCOM Compatibility:** This scale maps 1:1 to GEDCOM 5.5.1 QUAY values, ensuring lossless interoperability.

### Evidence Chain Completeness
A complete evidence chain requires:
1. **Repository**: Where the source is located
2. **Source**: The original document or record
3. **Citation**: Specific location within the source
4. **Assertion**: Conclusion drawn from the citation

```yaml
# repositories/repository-gro.glx
repositories:
  repository-gro:
    version: "1.0"
    name: General Register Office
    address: "London, England"

# sources/source-birth-register.glx
sources:
  source-birth-register:
    version: "1.0"
    title: England Birth Register 1850
    repository: repository-gro

# citations/citation-john-birth.glx
citations:
  citation-john-birth:
    version: "1.0"
    source: source-birth-register
    locator: "Volume 23, Page 145, Entry 23"
    quality: 3

# assertions/assertion-john-born.glx
assertions:
  assertion-john-born:
    version: "1.0"
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations: [citation-john-birth]
    confidence: high
```

## Provenance Tracking

### What is Provenance?
Provenance is the complete history of how information came to be known, including:
- **Source**: Where the information originated
- **Chain of Custody**: How it was preserved and transmitted
- **Author**: Who recorded or interpreted the information
- **Timestamp**: When the information was recorded
- **Context**: Circumstances surrounding the creation

### GENEALOGIX Provenance Features

#### 1. Source Attribution
Every assertion must reference specific citations:
```yaml
# assertions/assertion-smith-occupation.glx
assertions:
  assertion-smith-occupation:
    version: "1.0"
    subject: person-john-smith
    claim: occupation
    value: blacksmith
    citations:
      - citation-1851-census
      - citation-trade-directory
      - citation-parish-record
    created_by: researcher-jane-smith
    created_at: "2024-03-15T10:00:00Z"
```

#### 2. Change Tracking with Git
Since GENEALOGIX archives are Git repositories, all changes are automatically tracked:
```bash
# Git provides complete change history
git log --oneline -- persons/person-john-smith.glx

# See who made what changes
git blame persons/person-john-smith.glx

# Track research progress over time
git log --since="2024-01-01" --until="2024-03-31"
```

#### 3. Research Notes and Analysis
Structured fields for documenting research decisions:
```yaml
# assertions/assertion-disputed-birth.glx
assertions:
  assertion-disputed-birth:
    version: "1.0"
    subject: person-john-smith
    claim: birth_date
    value: "1850-01-15"
    confidence: medium
    research_notes: |
      Two conflicting sources:
      - Birth certificate: January 15, 1850 (preferred, higher quality)
      - Baptism record: January 20, 1850 (5-day delay common)

      Certificate takes precedence as primary direct evidence.
      More research needed on baptism delay practices.
    citations:
      - citation-birth-cert
      - citation-baptism-record
```

#### 4. Confidence Assessment
Assertions include confidence levels based on evidence quality:
```yaml
confidence_levels:
  high:    "Multiple primary sources agree"
  medium:  "Some conflicting evidence, but preponderance supports"
  low:     "Limited evidence, requires more research"
  disputed: "Multiple sources conflict, resolution unclear"
```

## Version Control Integration

### Git-Native Architecture
GENEALOGIX is designed from the ground up for Git version control:

#### 1. File-Based Structure
Each entity is a separate YAML file, perfect for Git tracking:
```
family-archive/
├── persons/
│   ├── person-john-smith.glx
│   ├── person-mary-brown.glx
│   └── person-jane-smith.glx
├── events/
│   ├── event-birth-john.glx
│   └── event-marriage.glx
└── citations/
    └── citation-1851-census.glx
```

#### 2. Branch-Based Research
Isolate research investigations in branches:
```bash
# Research specific time period
git checkout -b research/1851-census-analysis

# Add census data and analysis
# ... make changes ...

# Validate research branch
glx validate

# Merge findings when complete
git checkout main
git merge research/1851-census-analysis
```

#### 3. Collaborative Workflows
Multiple researchers can work simultaneously:
```bash
# Researcher A: Focus on vital records
git checkout -b research/vital-records
# ... work on birth, marriage, death records ...

# Researcher B: Focus on census data
git checkout -b research/census-records
# ... work on census information ...

# Integrate both research streams
git checkout main
git merge research/vital-records
git merge research/census-records  # May need conflict resolution
```

#### 4. Evidence Conflict Resolution
Git helps resolve conflicting evidence:
```bash
# Scenario: Two researchers find different birth dates
git merge research/birth-certificate-data
# CONFLICT: persons/person-john-smith.glx

# Manual resolution: Create assertion with both citations
# Git tracks the resolution process
git add persons/person-john-smith.glx
git commit -m "Resolve birth date conflict

Evidence:
- Birth certificate: Jan 15, 1850 (quality 3)
- Census record: age 25 in 1875 (implies 1850, quality 2)

Resolution: Accept certificate date, census supports approximately"
```

### Advanced Git Features

#### 1. Bisect for Research Errors
Find when incorrect information was introduced:
```bash
# Archive shows person born in wrong place
git bisect start
git bisect bad HEAD  # Current state has error
git bisect good v1.0.0  # Known good state

# Git finds the commit that introduced the error
git bisect run glx validate
```

#### 2. Stash for Experimentation
Try research hypotheses without committing:
```bash
# Start investigating alternative theory
git stash push -m "Trying alternative birth place theory"

# Make experimental changes
# ... modify files ...

# Validate hypothesis
glx validate

# If promising, commit; if not, restore
git stash pop  # Discard changes
# or
git stash pop && git add . && git commit  # Keep changes
```

#### 3. Interactive Rebase for Research History
Clean up research process:
```bash
# Clean up messy research commits
git rebase -i v1.0.0

# Combine related research steps
# Edit commit messages for clarity
# Remove dead-end investigations
```

## Entity Relationships

### Core Entity Connections
The 9 GENEALOGIX entity types form an interconnected web:

```
Person ←→ Relationship ←→ Person
Person ←→ Event ←→ Place
Source ←→ Citation → Assertion ←→ Person/Event/Place
Repository → Source
```

### Validation Dependencies
These relationships create validation requirements:
- Citations must reference existing sources
- Assertions must reference existing citations
- Events must reference existing places (if place specified)
- Participants must reference existing persons

### Reference Integrity
GENEALOGIX enforces referential integrity:
```yaml
# Valid: All referenced entities exist
events/event-wedding.glx:
  place: place-leeds-parish-church  # Must exist in places/
  participants:
    - person: person-john-smith     # Must exist in persons/
    - person: person-mary-brown     # Must exist in persons/

# Invalid: Referenced entities don't exist
# glx validate will catch these errors
```

This core concept architecture ensures that GENEALOGIX archives are reliable, verifiable, and maintainable over time.


