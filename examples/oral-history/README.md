# Oral History Example

A GENEALOGIX archive demonstrating the documentation of oral traditions,
family stories, and interviews as genealogical sources.

## Structure

```
oral-history/
├── .oracynth/
│   ├── config.glx
│   └── schema-version.glx
├── persons/
│   ├── person-storyteller.glx
│   ├── person-parent.glx
│   └── person-sibling.glx
├── relationships/
│   ├── rel-parent-elena.glx
│   ├── rel-sibling-elena.glx
│   └── rel-marriage-carlos.glx
├── sources/
│   ├── source-oral-interview.glx
│   └── source-family-stories.glx
├── media/
│   ├── media-interview-audio.glx
│   ├── media-interview-transcript.glx
│   └── media-family-stories.glx
├── assertions/
│   ├── assert-birth-elena.glx
│   ├── assert-immigration-elena.glx
│   ├── assert-birth-carlos.glx
│   ├── assert-immigration-carlos.glx
│   └── assert-birth-miguel.glx
└── README.md
```

## Family Overview

This example documents the Martinez family's oral history:

- **Carlos Martinez** (b. ~1930, Mexico) immigrated to Los Angeles in 1955
- **Elena Martinez** (b. 1965, Mexico) immigrated to Los Angeles in 1980
- **Miguel Martinez** (b. 1968, Los Angeles) is Elena's brother
- Family stories and traditions are preserved through oral interviews

## Key Concepts Demonstrated

### Oral History Documentation
- **Audio recordings**: Primary source interviews
- **Transcripts**: Written documentation of oral sources
- **Family stories**: Traditional knowledge and folklore
- **Multiple perspectives**: Different family members' accounts

### Source Evaluation
- **Medium confidence**: Oral history requires careful evaluation
- **Low confidence**: Information from memory without documentation
- **Source correlation**: Multiple family members confirming stories
- **Cultural context**: Understanding of oral tradition importance

### Immigration Documentation
- **Place of origin**: Birth location and cultural background
- **Migration patterns**: Family movement and settlement
- **Cultural preservation**: Maintaining traditions in new locations
- **Generational differences**: First vs. second generation experiences

## Files

### sources/source-oral-interview.glx
```yaml
id: source-oral-interview
version: "1.0"
title: "Oral History Interview - Elena Martinez"
type: "Oral History"
authors: ["Elena Martinez"]
date: "2024-01-15"
citation: "Personal interview with Elena Martinez, conducted by Maria Rodriguez, January 15, 2024, Los Angeles, California"

media:
  - media/media-interview-audio.glx
  - media/media-interview-transcript.glx
```

### assertions/assert-birth-elena.glx
```yaml
id: assert-birth-elena
version: "1.0"
type: birth
subject: person-storyteller
date: "1965-07-20"
place: "Guadalajara, Mexico"
confidence: medium

sources:
  - sources/source-oral-interview.glx
  - sources/source-family-stories.glx
```

### media/media-interview-audio.glx
```yaml
id: media-interview-audio
version: "1.0"
uri: "media/elena-martinez-interview-2024-01-15.mp3"
mime_type: "audio/mpeg"
hash: "sha256:oral123history456..."
```

## Validation

```bash
glx validate
# ✓ All files valid

glx check-schemas
# ✓ schemas valid
```

## What This Demonstrates

- **Oral tradition preservation**: Capturing family stories and cultural knowledge
- **Audio documentation**: Primary source interviews with proper citation
- **Cultural context**: Understanding immigration and cultural preservation
- **Source evaluation**: Appropriate confidence levels for oral sources
- **Media management**: Audio files, transcripts, and document preservation
- **Family relationships**: Documenting relationships through oral history

## Oral History Best Practices

This example follows oral history methodology:

1. **Informed consent**: Clear documentation of interview permissions
2. **Proper citation**: Full attribution of sources and interviewers
3. **Media preservation**: Audio recordings with transcript backups
4. **Cultural sensitivity**: Respect for family traditions and privacy
5. **Source evaluation**: Appropriate confidence levels for oral sources

## Cultural Significance

Oral history is particularly important for:

- **Immigrant families**: Documenting migration stories and cultural preservation
- **Marginalized communities**: Preserving stories not found in official records
- **Cultural traditions**: Maintaining family customs and folklore
- **Generational knowledge**: Passing down stories and experiences

## Next Steps

Expand the oral history collection:
- Interview additional family members
- Document cultural traditions and recipes
- Record family language and dialect preservation
- Create video interviews for visual documentation
- Transcribe interviews in multiple languages
