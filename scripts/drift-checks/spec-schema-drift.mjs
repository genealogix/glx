#!/usr/bin/env node
// Spec <-> schema field-parity check (issue #309).
//
// Compares the field tables in specification/4-entity-types/*.md against the
// properties of specification/schema/v1/*.schema.json. The dangerous case it
// catches: a field documented in the spec but absent from a schema whose
// additionalProperties is false — the validator then rejects data the spec
// says is valid.
//
// WARN-FIRST: this is deterministic but the markdown-table parser can have
// edge cases, so it reports and exits 0 by default. Set DRIFT_STRICT=1 to make
// it exit non-zero on any mismatch (flip the CI job to blocking once proven).

import { readFileSync, readdirSync } from "fs";
import { join, dirname, basename } from "path";
import { fileURLToPath } from "url";

const ROOT = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const SPEC_DIR = join(ROOT, "specification/4-entity-types");
const SCHEMA_DIR = join(ROOT, "specification/schema/v1");

// Field names that appear in the spec field tables but are structural notes,
// not entity properties (e.g. the map-key row).
const NON_FIELD_ROWS = new Set(["entity id (map key)"]);

// Parse the TOP-LEVEL field tables and return the set of backtick-wrapped
// field names. Across the entity specs the top-level entity fields always sit
// under a `### Required Fields` / `### Optional Fields` heading; nested
// sub-object tables (Subject Object, Participant Object Fields, Search entry
// fields, …) sit under other headings and must NOT be treated as entity
// properties — scoping to the two headings avoids that whole false-positive
// class.
const TOP_LEVEL_FIELD_HEADINGS = new Set(["required fields", "optional fields"]);

function parseSpecFields(md) {
  // rowFields: field named in the first column (precise — the field is a row).
  // mentioned: every backtick token anywhere in a field-table row, so a
  // combined row like "| Evidence reference | array | one of `citations`,
  // `sources`, or `media` |" still counts those three as documented.
  const rowFields = new Set();
  const mentioned = new Set();
  const lines = md.split("\n");
  let underTopLevel = false;
  let inTable = false;
  for (const raw of lines) {
    const line = raw.trim();
    const heading = line.match(/^#{2,4}\s+(.*)$/);
    if (heading) {
      underTopLevel = TOP_LEVEL_FIELD_HEADINGS.has(
        heading[1].trim().toLowerCase().replace(/[`*]/g, ""),
      );
      inTable = false;
      continue;
    }
    if (!underTopLevel) continue;
    if (/^\|\s*Field\s*\|\s*Type\s*\|/i.test(line)) {
      inTable = true;
      continue;
    }
    if (!inTable) continue;
    if (!line.startsWith("|")) {
      inTable = false;
      continue;
    }
    if (/^\|[\s:-]+\|/.test(line)) continue; // separator row
    for (const tok of line.matchAll(/`([^`]+)`/g)) mentioned.add(tok[1]);
    const firstCell = (line.split("|")[1] || "").trim();
    if (NON_FIELD_ROWS.has(firstCell.toLowerCase())) continue;
    const m = firstCell.match(/^`([^`]+)`$/);
    if (m) rowFields.add(m[1]);
  }
  return { rowFields, mentioned };
}

function schemaProps(schema) {
  return new Set(Object.keys(schema.properties || {}));
}

function diff(a, b) {
  return [...a].filter((x) => !b.has(x)).sort();
}

const specFiles = readdirSync(SPEC_DIR)
  .filter((f) => f.endsWith(".md") && f !== "README.md");

let mismatches = 0;
let checked = 0;

for (const file of specFiles) {
  const stem = basename(file, ".md");
  const schemaPath = join(SCHEMA_DIR, `${stem}.schema.json`);
  let schema;
  try {
    schema = JSON.parse(readFileSync(schemaPath, "utf8"));
  } catch {
    console.warn(`⚠️  ${stem}: spec file has no matching schema (${stem}.schema.json) — skipped`);
    continue;
  }
  checked++;

  const { rowFields, mentioned } = parseSpecFields(readFileSync(join(SPEC_DIR, file), "utf8"));
  const schemaFields = schemaProps(schema);
  const apFalse = schema.additionalProperties === false;

  // Spec documents a field as its own row, but the schema lacks it.
  const inSpecNotSchema = diff(rowFields, schemaFields);
  // Schema has a field the spec never mentions in its field tables (combined
  // rows count, so this is genuine "undocumented", not a formatting artifact).
  const inSchemaNotSpec = diff(schemaFields, mentioned);

  if (inSpecNotSchema.length === 0 && inSchemaNotSpec.length === 0) {
    console.log(`✓ ${stem}: ${schemaFields.size} fields match`);
    continue;
  }

  for (const f of inSpecNotSchema) {
    console.error(
      `✗ ${stem}: field \`${f}\` is documented in the spec but missing from the schema` +
        (apFalse ? " (additionalProperties:false → the validator rejects it)" : ""),
    );
    mismatches++;
  }
  for (const f of inSchemaNotSpec) {
    console.error(`✗ ${stem}: field \`${f}\` is in the schema but undocumented in the spec`);
    mismatches++;
  }
}

console.log(
  `\nspec↔schema parity: ${checked} entity types checked, ${mismatches} field mismatch(es).`,
);

if (mismatches > 0 && process.env.DRIFT_STRICT === "1") {
  process.exit(1);
}
