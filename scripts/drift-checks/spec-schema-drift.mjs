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
  // requiredFields / optionalFields: first-column field names, split by whether
  // the row sits under "### Required Fields" or "### Optional Fields" — this is
  // what lets us compare required-vs-optional against the schema's required[].
  // mentioned: every backtick token anywhere in a field-table row, so a combined
  // row like "| Evidence reference | array | one of `citations`, `sources`, or
  // `media` |" still counts those three as documented.
  const requiredFields = new Set();
  const optionalFields = new Set();
  const mentioned = new Set();
  const lines = md.split("\n");
  let section = null; // "required" | "optional" | null
  let inTable = false;
  for (const raw of lines) {
    const line = raw.trim();
    const heading = line.match(/^#{2,4}\s+(.*)$/);
    if (heading) {
      const h = heading[1].trim().toLowerCase().replace(/[`*]/g, "");
      section = TOP_LEVEL_FIELD_HEADINGS.has(h) ? h.split(" ")[0] : null; // "required"|"optional"
      inTable = false;
      continue;
    }
    if (!section) continue;
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
    if (m) (section === "required" ? requiredFields : optionalFields).add(m[1]);
  }
  return { requiredFields, optionalFields, mentioned };
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

  const { requiredFields, optionalFields, mentioned } = parseSpecFields(
    readFileSync(join(SPEC_DIR, file), "utf8"),
  );
  const rowFields = new Set([...requiredFields, ...optionalFields]);
  const schemaFields = schemaProps(schema);
  const schemaRequired = new Set(schema.required || []);
  const apFalse = schema.additionalProperties === false;

  // Spec documents a field as its own row, but the schema lacks it.
  const inSpecNotSchema = diff(rowFields, schemaFields);
  // Schema has a field the spec never mentions in its field tables (combined
  // rows count, so this is genuine "undocumented", not a formatting artifact).
  const inSchemaNotSpec = diff(schemaFields, mentioned);
  // Required/optional drift (only for fields present in both).
  const specRequiredNotSchema = [...requiredFields]
    .filter((f) => schemaFields.has(f) && !schemaRequired.has(f)).sort();
  const specOptionalButSchemaRequired = [...optionalFields]
    .filter((f) => schemaRequired.has(f)).sort();

  if (
    inSpecNotSchema.length === 0 && inSchemaNotSpec.length === 0 &&
    specRequiredNotSchema.length === 0 && specOptionalButSchemaRequired.length === 0
  ) {
    console.log(`✓ ${stem}: ${schemaFields.size} fields match (presence + required/optional)`);
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
  for (const f of specRequiredNotSchema) {
    console.error(`✗ ${stem}: field \`${f}\` is under "Required Fields" in the spec but is not in the schema's required[]`);
    mismatches++;
  }
  for (const f of specOptionalButSchemaRequired) {
    console.error(`✗ ${stem}: field \`${f}\` is under "Optional Fields" in the spec but the schema marks it required`);
    mismatches++;
  }
}

console.log(
  `\nspec↔schema parity: ${checked} entity types checked, ${mismatches} field mismatch(es).`,
);

if (mismatches > 0 && process.env.DRIFT_STRICT === "1") {
  process.exit(1);
}
