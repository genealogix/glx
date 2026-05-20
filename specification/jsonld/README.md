# JSON-LD context

`glx-context.jsonld` is the canonical JSON-LD `@context` for the GENEALOGIX archive format. It maps GLX entity types and common properties to [Schema.org](https://schema.org/) classes (`Person`, `Event`, `Place`, `CreativeWork`, `ArchiveOrganization`, `MediaObject`, `GeoCoordinates`, `PostalAddress`) and uses a `glx:` namespace for GLX-native concepts that Schema.org does not cover (`Citation`, `Relationship`, `Assertion`, `Participation`, `VocabularyEntry`).

This file is the source of truth. It is embedded into the `go-glx` library and inlined into every `glx export --format jsonld` output so emitted documents are self-contained — no network access required to resolve the context.

See [issue #291](https://github.com/genealogix/glx/issues/291) for the design discussion.
