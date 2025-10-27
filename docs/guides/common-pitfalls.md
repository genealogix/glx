---
title: Common Pitfalls
description: Avoid common mistakes when working with GENEALOGIX archives
layout: doc
---

# Common Pitfalls Guide

This guide helps you avoid the most frequent mistakes when working with GENEALOGIX archives.

## Evidence and Citation Issues

### 1. Missing Evidence Chains

**Problem:** Making claims without evidence
```yaml
# ❌ Wrong: Claim without evidence
persons/person-john-smith.glx:
  birth: "1850-01-15"  # No source!

# ✅ Correct: Evidence-based approach
assertions/assertion-birth.glx:
  subject: person-john-smith
  claim: born_on
  value: "1850-01-15"
  citations: [citation-birth-cert]
```

**Why this matters:**
- No audit trail from source to conclusion
- Impossible to verify accuracy
- Can't resolve conflicting evidence
- No quality assessment possible

### 2. Incomplete Evidence Chains

**Problem:** Broken reference chains
```yaml
# ❌ Wrong: Citation without source
citations/citation-missing-source.glx:
  source: source-nonexistent  # Source doesn't exist!
  locator: "Page 23"

# ✅ Correct: Complete chain
citations/citation-good.glx:
  source: source-birth-register  # Source exists
  locator: "Entry 145, Page 23"
```

**Validation will catch this:**
```bash
glx validate
# ERROR: citation-missing-source.glx references nonexistent source
```

### 3. Quality Rating Mistakes

**Problem:** Incorrect evidence quality assessment
```yaml
# ❌ Wrong: Census as primary evidence
citations/citation-census-primary.glx:
  source: source-1851-census
  quality: 3  # Too high for census!
  notes: "This is primary evidence"

# ✅ Correct: Census as secondary evidence
citations/citation-census-secondary.glx:
  source: source-1851-census
  quality: 2  # Correct for census
  notes: "Secondary source - created after the event"
```

**Quality guidelines:**
- **3**: Original records (birth cert, baptism)
- **2**: Records created later (census, marriage index)
- **1**: Family records (Bible, letters)
- **0**: Compiled records (published genealogy)

## File Organization Issues

### 1. Wrong Directory Structure

**Problem:** Files in wrong directories
```bash
# ❌ Wrong: Person file in events directory
events/person-john-smith.glx  # Should be in persons/

# ✅ Correct: Proper directory structure
persons/person-john-smith.glx
events/event-john-birth.glx
```

**Problem:** Mixed entity types
```bash
# ❌ Wrong: Multiple types in one directory
persons/
├── person-john.glx
├── event-birth.glx    # Wrong directory!
└── place-leeds.glx    # Wrong directory!

# ✅ Correct: Each type in its own directory
persons/person-john.glx
events/event-birth.glx
places/place-leeds.glx
```

### 2. Incorrect ID Formats

**Problem:** Non-standard ID formats
```yaml
# ❌ Wrong: Human-readable IDs
id: john-smith-person
id: birth-of-john-smith
id: leeds-england

# ✅ Correct: Structured IDs
id: person-a1b2c3d4
id: event-b2c3d4e5
id: place-c3d4e5f6
```

**Problem:** Missing type prefixes
```yaml
# ❌ Wrong: ID without type prefix
id: a1b2c3d4  # Missing "person-" prefix!

# ✅ Correct: Full type prefix
id: person-a1b2c3d4  # Includes "person-" prefix
```

### 3. File Extension Errors

**Problem:** Wrong file extensions
```bash
# ❌ Wrong: Non-standard extensions
persons/person-john.txt
events/birth.yaml
places/leeds.json

# ✅ Correct: .glx extension
persons/person-john.glx
events/event-birth.glx
places/place-leeds.glx
```

## YAML Syntax Issues

### 1. Indentation Problems

**Problem:** Inconsistent indentation
```yaml
# ❌ Wrong: Mixed tabs and spaces
persons/person-john.glx:
  id: person-john
   name: "John"  # Inconsistent indentation

# ✅ Correct: Consistent spacing
persons/person-john.glx:
  id: person-john
  name: "John"
```

**Problem:** Wrong indentation level
```yaml
# ❌ Wrong: Incorrect nesting
name:
given: John
 surname: Smith  # Wrong indentation

# ✅ Correct: Proper nesting
name:
  given: John
  surname: Smith
```

### 2. Quoting Issues

**Problem:** Inconsistent or missing quotes
```yaml
# ❌ Wrong: Inconsistent quoting
name: John Smith  # Unquoted
notes: "This has quotes"  # Quoted

# ✅ Correct: Consistent quoting for special characters
name: "John Smith"
notes: "Contains special chars: é, ñ, 中文"
```

**Problem:** Missing quotes for special values
```yaml
# ❌ Wrong: Missing quotes
date: 1850-01-15  # Should be quoted
quality: 3  # Should be quoted

# ✅ Correct: Quoted values
date: "1850-01-15"
quality: 3
```

### 3. Boolean and Null Values

**Problem:** Incorrect boolean/null syntax
```yaml
# ❌ Wrong: Python-style booleans
verified: True
spouse: None

# ✅ Correct: YAML booleans and nulls
verified: true
spouse: null
```

## Reference Issues

### 1. Broken References

**Problem:** Nonexistent entity references
```yaml
# ❌ Wrong: References don't exist
events/event-wedding.glx:
  place: place-nonexistent  # Place doesn't exist
  participants:
    - person: person-missing  # Person doesn't exist

# ✅ Correct: All references exist
events/event-wedding.glx:
  place: place-leeds-parish-church  # Place exists
  participants:
    - person: person-john-smith  # Person exists
```

### 2. Circular References

**Problem:** Self-referencing or circular references
```yaml
# ❌ Wrong: Self-reference
places/place-leeds.glx:
  parent: place-leeds  # Can't be parent of itself!

# ✅ Correct: Proper hierarchy
places/place-leeds.glx:
  parent: place-yorkshire  # Parent exists and is different
```

### 3. Missing Required References

**Problem:** Missing required entity references
```yaml
# ❌ Wrong: Event without place
events/event-birth.glx:
  event_type: birth
  date: "1850-01-15"
  # Missing place reference!

# ✅ Correct: Place reference included
events/event-birth.glx:
  event_type: birth
  date: "1850-01-15"
  place: place-leeds
```

## Validation Issues

### 1. Schema Validation Errors

**Problem:** Files don't match JSON Schema
```yaml
# ❌ Wrong: Invalid field names
persons/person-john.glx:
  id: person-john
  name: "John Smith"
  birthdate: "1850-01-15"  # Wrong field name!

# ✅ Correct: Valid field names
persons/person-john.glx:
  id: person-john
  name: "John Smith"
  birth: "1850-01-15"  # Correct field name
```

**Check schema requirements:**
```bash
# See what fields are required
glx check-schemas

# Validate against schema
glx validate persons/person-john.glx
```

### 2. Pattern Validation Errors

**Problem:** Values don't match required patterns
```yaml
# ❌ Wrong: Invalid ID pattern
id: person-john-smith  # Should be person-{8hex}

# ✅ Correct: Valid ID pattern
id: person-a1b2c3d4  # Matches pattern
```

**Problem:** Invalid date formats
```yaml
# ❌ Wrong: Non-standard date formats
date: 01/15/1850  # US format
date: 15 Jan 1850  # Text format

# ✅ Correct: ISO format
date: "1850-01-15"
```

## Git Workflow Issues

### 1. Large Commits

**Problem:** Too many changes in one commit
```bash
# ❌ Wrong: Massive commit
git add .
git commit -m "big update"

# ✅ Correct: Focused commits
git add persons/
git commit -m "Add John Smith person record

- Basic biographical information
- Birth and death dates
- Occupation as blacksmith"

git add events/
git commit -m "Add Smith family events

- Birth events for 3 children
- Marriage event with citation
- Death events with sources"
```

### 2. Poor Commit Messages

**Problem:** Unclear commit messages
```bash
# ❌ Wrong: Vague messages
git commit -m "update"
git commit -m "changes"
git commit -m "fix"

# ✅ Correct: Descriptive messages
git commit -m "Add 1851 Census evidence for Smith family

Source: HO107, Piece 2319, Yorkshire
Added evidence for:
- John Smith: occupation, residence
- Mary Smith: birthplace, age
- Jane Smith: school attendance

Quality: 2 (secondary source)
All citations reference existing sources"
```

### 3. Unvalidated Commits

**Problem:** Committing without validation
```bash
# ❌ Wrong: Commit without validation
git add .
git commit -m "add data"

# ✅ Correct: Validate first
glx validate
# Fix any validation errors
git add .
git commit -m "add validated data"
```

## Data Quality Issues

### 1. Conflicting Evidence Without Resolution

**Problem:** Multiple conflicting claims
```yaml
# ❌ Wrong: Conflicting assertions without resolution
assertions/assertion-birth-1.glx:
  claim: born_on
  value: "1850-01-15"
  citations: [citation-cert]

assertions/assertion-birth-2.glx:
  claim: born_on
  value: "1850-01-20"  # Conflicts!
  citations: [citation-baptism]

# ✅ Correct: Single resolved assertion
assertions/assertion-birth-resolved.glx:
  claim: born_on
  value: "1850-01-15"  # Resolved value
  confidence: medium
  research_notes: |
    Conflicting evidence resolved:
    - Birth cert: Jan 15 (quality 3) - preferred
    - Baptism: Jan 20 (quality 3) - 5 day delay common
  citations: [citation-cert, citation-baptism]
```

### 2. Place Hierarchy Errors

**Problem:** Invalid place relationships
```yaml
# ❌ Wrong: Circular or invalid hierarchy
places/place-leeds.glx:
  parent: place-leeds  # Self-reference!

places/place-yorkshire.glx:
  parent: place-leeds  # Child can't be parent of parent!

# ✅ Correct: Proper hierarchy
places/place-england.glx:
  name: England
  type: country

places/place-yorkshire.glx:
  name: Yorkshire
  type: county
  parent: place-england

places/place-leeds.glx:
  name: Leeds
  type: city
  parent: place-yorkshire
```

### 3. Event Participant Issues

**Problem:** Missing or invalid participants
```yaml
# ❌ Wrong: Event without participants
events/event-marriage.glx:
  event_type: marriage
  date: "1875-05-10"
  # Missing participants!

# ✅ Correct: Proper participants
events/event-marriage.glx:
  event_type: marriage
  date: "1875-05-10"
  place: place-leeds-parish-church
  participants:
    - person: person-john-smith
      role: groom
    - person: person-mary-brown
      role: bride
```

## Performance Issues

### 1. Large Files

**Problem:** Overly complex single files
```yaml
# ❌ Wrong: Massive person file
persons/person-john.glx:
  # 1000+ lines with everything

# ✅ Correct: Split into focused files
persons/person-john.glx:
  # Basic info only

events/event-john-birth.glx:
  # Birth event

events/event-john-marriage.glx:
  # Marriage event

events/event-john-occupation.glx:
  # Occupation events
```

### 2. Duplicate Data

**Problem:** Same information in multiple places
```yaml
# ❌ Wrong: Duplicate place info
persons/person-john.glx:
  birth_place: "Leeds, Yorkshire, England"

places/place-leeds.glx:
  name: "Leeds, Yorkshire, England"  # Duplicate!

# ✅ Correct: Single source of truth
persons/person-john.glx:
  birth_place: place-leeds  # Reference only

places/place-leeds.glx:
  name: "Leeds"
  type: city
  parent: place-yorkshire
```

## Troubleshooting Checklist

### Before Committing
- [ ] Run `glx validate` on all files
- [ ] Check for broken references
- [ ] Verify evidence chains are complete
- [ ] Test with example files: `glx validate examples/`

### When Validation Fails
1. **Check file extensions**: Must be `.glx`
2. **Verify directory structure**: Files in correct directories
3. **Validate ID formats**: Match `{type}-{8hex}` pattern
4. **Check references**: All referenced entities exist
5. **Review schema**: Fields match JSON Schema requirements

### When Git Conflicts Occur
1. **Understand the conflict**: What evidence is conflicting?
2. **Evaluate source quality**: Which has higher quality rating?
3. **Document resolution**: Explain why one source was preferred
4. **Create resolved assertion**: With research notes

### Performance Issues
- **Large archives**: Consider splitting by time periods or branches
- **Slow validation**: Validate directories separately
- **Git performance**: Use `.gitignore` for temporary files
- **Storage**: Use `git gc` periodically for repository optimization

## Quick Fixes

### Common Validation Errors
```bash
# File extension wrong
ERROR: file.txt is not a .glx file
FIX: Rename to file.glx

# Wrong directory
ERROR: person.glx found in events/
FIX: Move to persons/ directory

# Invalid ID format
ERROR: id "john-smith" doesn't match pattern
FIX: Change to "person-a1b2c3d4"

# Broken reference
ERROR: person "missing" not found
FIX: Create missing person or fix reference
```

### YAML Syntax Fixes
```bash
# Check YAML syntax
python -c "import yaml; yaml.safe_load(open('file.glx'))"

# Fix common issues
# - Use spaces, not tabs
# - Quote special characters
# - Match indentation levels
# - Use true/false/null (not True/False/None)
```

Following these guidelines will help you avoid the most common mistakes and maintain high-quality GENEALOGIX archives.
