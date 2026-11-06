#!/usr/bin/env node
// Spec <-> schema field-parity check (issue #309).
//
// Compares the field tables in specification/4-entity-types/*.md against the
// properties of specification/schema/v1/*.schema.json. The dangerous case it
// catches: a field documented in the spec but absent from a schema whose
// additionalProperties is false — the validator then rejects data the spec
// says is valid.
//
// `make check-spec-schema-drift` runs this with DRIFT_STRICT=1, both in the
// required validate-schemas job and in `make check`, so drift fails that merge
// gate (#309). A bare `node` run reports and exits 0. The parser and filesystem
// scan are covered by spec-schema-drift.test.mjs.
//
// The parsing/comparison core (parseSpecFields, compareEntity) has no I/O.
// scanTree accepts fixture directories and a logger for integration tests.

import { readFileSync, readdirSync, realpathSync } from "fs";
import { join, dirname, basename } from "path";
import { fileURLToPath } from "url";

const ROOT = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const SPEC_DIR = join(ROOT, "specification/4-entity-types");
const SCHEMA_DIR = join(ROOT, "specification/schema/v1");

// Field names that appear in the spec field tables but are structural notes,
// not entity properties (e.g. the map-key row).
const NON_FIELD_ROWS = new Set(["entity id (map key)"]);

// Spec pages under 4-entity-types/ that legitimately have no
// <stem>.schema.json. Anything else missing a schema is drift, not an
// exemption: since the check became blocking (#309) a skipped entity would be
// a silent hole — every documented field unvalidated, which is exactly the
// `additionalProperties:false` hazard this script exists to catch.
const SPEC_PAGES_WITHOUT_SCHEMA = new Set(["vocabularies"]);

// Schemas that legitimately have no page under 4-entity-types/: glx-file is the
// archive container, documented elsewhere, not an entity type. Any other schema
// without a spec page is drift — an entity the validator accepts but nobody
// documented.
const SCHEMAS_WITHOUT_SPEC_PAGE = new Set(["glx-file"]);

// Parse the TOP-LEVEL field tables and return the set of backtick-wrapped
// field names. Across the entity specs the top-level entity fields always sit
// under a `### Required Fields` / `### Optional Fields` heading; nested
// sub-object tables (Subject Object, Participant Object Fields, Search entry
// fields, …) sit under other headings and must NOT be treated as entity
// properties — scoping to the two headings avoids that whole false-positive
// class.
const TOP_LEVEL_FIELD_HEADINGS = new Set(["required fields", "optional fields"]);

export function parseSpecFields(md) {
  // requiredFields / optionalFields: first-column field names, split by whether
  // the row sits under "### Required Fields" or "### Optional Fields" — this is
  // what lets us compare required-vs-optional against the schema's required[].
  // A first cell may carry an annotation ("`label` (deprecated)"); its backticked
  // names are still the row's fields.
  // mentioned: the row fields, plus every backtick token in a combined row, one
  // whose first cell names no field: "| Evidence reference | array | one of
  // `citations`, `sources`, or `media` |" documents those three. A token in the
  // description of a named field's own row is prose, and documents nothing.
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
    const firstCell = (line.split("|")[1] || "").trim();
    if (NON_FIELD_ROWS.has(firstCell.toLowerCase())) continue;
    const rowNames = [...firstCell.matchAll(/`([^`]+)`/g)].map((tok) => tok[1]);
    if (rowNames.length === 0) {
      for (const tok of line.matchAll(/`([^`]+)`/g)) mentioned.add(tok[1]);
      continue;
    }
    const fields = section === "required" ? requiredFields : optionalFields;
    for (const name of rowNames) {
      fields.add(name);
      mentioned.add(name);
    }
  }
  return { requiredFields, optionalFields, mentioned };
}

export function schemaProps(schema) {
  return new Set(Object.keys(schema.properties || {}));
}

export function diff(a, b) {
  return [...a].filter((x) => !b.has(x)).sort();
}

// compareEntity is the testable core: given the spec markdown and the parsed
// schema for one entity, it returns the five drift classes (each a sorted
// array) plus apFalse, so callers can phrase the additionalProperties:false
// consequence. No I/O — every input is in memory.
export function compareEntity(specMd, schema) {
  const { requiredFields, optionalFields, mentioned } = parseSpecFields(specMd);
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
  // A required[] entry with no row of its own: a property only mentioned in a
  // combined row, or a name absent from properties, which makes an
  // additionalProperties:false schema reject every document.
  const schemaRequiredNotSpec = [...schemaRequired]
    .filter((f) => !requiredFields.has(f) && !optionalFields.has(f)).sort();

  return {
    inSpecNotSchema,
    inSchemaNotSpec,
    specRequiredNotSchema,
    specOptionalButSchemaRequired,
    schemaRequiredNotSpec,
    apFalse,
    schemaFieldCount: schemaFields.size,
  };
}

// Walk a spec directory and its schema directory and report the drift found.
// The directories are parameters (not the module constants) so the test file
// can drive the whole scan — including the pairing rules below — against
// fixture trees; run() supplies the real ones.
export function scanTree({ specDir = SPEC_DIR, schemaDir = SCHEMA_DIR, log = console } = {}) {
  let mismatches = 0;
  let checked = 0;
  let specFiles;
  let schemaFiles;
  const report = () => {
    log.log(
      `\nspec↔schema parity: ${checked} entity types checked, ${mismatches} drift finding(s).`,
    );
    return { checked, mismatches };
  };

  try {
    specFiles = readdirSync(specDir)
      .filter((f) => f.endsWith(".md") && f !== "README.md");
  } catch (err) {
    log.error(`✗ cannot read the spec directory (${specDir}): ${err.code || err.message}`);
    mismatches++;
  }
  try {
    schemaFiles = readdirSync(schemaDir)
      .filter((f) => f.endsWith(".schema.json"));
  } catch (err) {
    log.error(`✗ cannot read the schema directory (${schemaDir}): ${err.code || err.message}`);
    mismatches++;
  }

  // If either directory cannot be listed, individual pairing claims would be
  // misleading. The directory failure already counts as drift.
  if (!specFiles || !schemaFiles) return report();

  // Every stem that has a spec page, whether or not its schema loaded — the
  // orphan sweep below must not double-report a schema already reported as
  // unreadable.
  const documentedStems = new Set();

  for (const file of specFiles) {
    const stem = basename(file, ".md");
    documentedStems.add(stem);
    let specMd;
    try {
      specMd = readFileSync(join(specDir, file), "utf8");
    } catch (err) {
      log.error(`✗ ${stem}: cannot read spec page (${file}: ${err.code || err.message}) — its fields are unchecked`);
      mismatches++;
      continue;
    }
    const schemaPath = join(schemaDir, `${stem}.schema.json`);
    let raw;
    try {
      raw = readFileSync(schemaPath, "utf8");
    } catch (err) {
      // Only an absent file can be an allowlisted "this page has no schema".
      // A read error on a schema that exists is drift even for an allowlisted
      // page — otherwise a broken vocabularies.schema.json would slip through.
      if (err.code === "ENOENT" && SPEC_PAGES_WITHOUT_SCHEMA.has(stem)) {
        log.warn(`⚠️  ${stem}: no schema expected for this page — skipped`);
      } else {
        log.error(
          `✗ ${stem}: spec page has no readable schema (${stem}.schema.json: ${err.code || err.message}) — its documented fields are unchecked`,
        );
        mismatches++;
      }
      continue;
    }

    let schema;
    try {
      schema = JSON.parse(raw);
    } catch (err) {
      log.error(`✗ ${stem}: ${stem}.schema.json is not valid JSON (${err.message})`);
      mismatches++;
      continue;
    }
    if (
      schema === null || typeof schema !== "object" || Array.isArray(schema) ||
      (schema.properties !== undefined &&
        (schema.properties === null || typeof schema.properties !== "object" || Array.isArray(schema.properties))) ||
      (schema.required !== undefined &&
        (!Array.isArray(schema.required) || !schema.required.every((field) => typeof field === "string")))
    ) {
      log.error(`✗ ${stem}: ${stem}.schema.json is not an entity schema object with valid properties and required fields`);
      mismatches++;
      continue;
    }

    const r = compareEntity(specMd, schema);
    const clean =
      r.inSpecNotSchema.length === 0 && r.inSchemaNotSpec.length === 0 &&
      r.specRequiredNotSchema.length === 0 && r.specOptionalButSchemaRequired.length === 0 &&
      r.schemaRequiredNotSpec.length === 0;

    // No fields on either side agree vacuously: nothing was verified, so an
    // empty page paired with a `{}` or allOf-only schema must not count.
    if (clean && r.schemaFieldCount === 0) {
      log.error(`✗ ${stem}: neither the spec page nor ${stem}.schema.json defines a field — nothing was compared`);
      mismatches++;
      continue;
    }
    checked++;

    if (clean) {
      log.log(`✓ ${stem}: ${r.schemaFieldCount} fields match (presence + required/optional)`);
      continue;
    }

    for (const f of r.inSpecNotSchema) {
      log.error(
        `✗ ${stem}: field \`${f}\` is documented in the spec but missing from the schema` +
          (r.apFalse ? " (additionalProperties:false → the validator rejects it)" : ""),
      );
      mismatches++;
    }
    for (const f of r.inSchemaNotSpec) {
      log.error(`✗ ${stem}: field \`${f}\` is in the schema but undocumented in the spec`);
      mismatches++;
    }
    for (const f of r.specRequiredNotSchema) {
      log.error(`✗ ${stem}: field \`${f}\` is under "Required Fields" in the spec but is not in the schema's required[]`);
      mismatches++;
    }
    for (const f of r.specOptionalButSchemaRequired) {
      log.error(`✗ ${stem}: field \`${f}\` is under "Optional Fields" in the spec but the schema marks it required`);
      mismatches++;
    }
    for (const f of r.schemaRequiredNotSpec) {
      log.error(`✗ ${stem}: field \`${f}\` is in the schema's required[] but has no row under "Required Fields" in the spec`);
      mismatches++;
    }
  }

  // The loop above is driven by spec pages, so a schema with no page of its own
  // would never be looked at: an entity the validator accepts but nobody
  // documented. Sweep the schema directory for those too.
  for (const file of schemaFiles) {
    const stem = basename(file, ".schema.json");
    if (documentedStems.has(stem) || SCHEMAS_WITHOUT_SPEC_PAGE.has(stem)) continue;
    log.error(
      `✗ ${stem}: ${file} has no spec page (${stem}.md) — its fields are undocumented`,
    );
    mismatches++;
  }

  // Do not report an empty run on top of a specific read or parse failure.
  if (checked === 0 && mismatches === 0) {
    log.error(
      `✗ no entity spec/schema pairs found under ${specDir} — the check verified nothing`,
    );
    mismatches++;
  }

  return report();
}

// The strict gate itself: drift fails the process only under DRIFT_STRICT=1,
// which `make check-spec-schema-drift` sets. Exported so the exit policy is
// unit-tested rather than implied by the one call site.
export function exitCodeFor(mismatches, env = process.env) {
  return mismatches > 0 && env.DRIFT_STRICT === "1" ? 1 : 0;
}

export function run({ env = process.env, ...options } = {}) {
  const { mismatches } = scanTree(options);
  return exitCodeFor(mismatches, env);
}

// True when `scriptPath` (process.argv[1]) is this module. Both sides are
// canonicalized: node resolves links in import.meta.url but not in argv[1], so
// comparing them raw makes a run through a symlink or junction skip the scan
// and exit 0, a silent pass for the blocking gate.
export function isEntrypoint(scriptPath, moduleUrl = import.meta.url) {
  if (!scriptPath) return false;
  return realpathSync.native(scriptPath) === realpathSync.native(fileURLToPath(moduleUrl));
}

// Run the filesystem walk only when invoked as a script, not when imported by
// the test file (which calls the exported pure functions against fixtures).
if (isEntrypoint(process.argv[1])) {
  process.exitCode = run();
}
