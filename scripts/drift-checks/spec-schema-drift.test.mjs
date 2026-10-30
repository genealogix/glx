// Parser and filesystem tests for the spec<->schema field-parity gate (#309).
//
// The parser is a hand-rolled markdown-table reader. These fixtures pin its
// section scoping, structural rows, combined-row tokens, and drift classes.
//
// Run: node --test scripts/drift-checks/spec-schema-drift.test.mjs

import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  parseSpecFields,
  compareEntity,
  scanTree,
  exitCodeFor,
  run,
} from "./spec-schema-drift.mjs";

// Build markdown from lines so backticks stay literal (single-quoted strings).
const md = (...lines) => lines.join("\n");

// A representative top-level entity spec: Required + Optional field tables in the
// real format, a structural map-key row, a combined "one of ..." row, and a
// nested sub-object table that must be ignored.
const SPEC = md(
  "# Thing Entity",
  "",
  "## Fields",
  "",
  "### Required Fields",
  "",
  "| Field | Type | Description |",
  "| ----- | ---- | ----------- |",
  "| Entity ID (map key) | string | structural note, not a property |",
  "| `id` | string | the identifier |",
  "",
  "### Optional Fields",
  "",
  "| Field | Type | Description |",
  "| ----- | ---- | ----------- |",
  "| `label` | string | display label |",
  "| `evidence` | array | one of `citations`, `sources`, or `media` |",
  "",
  "### Subject Object",
  "",
  "| Field | Type | Description |",
  "| ----- | ---- | ----------- |",
  "| `nested_only` | string | belongs to a sub-object, NOT a top-level field |",
);

test("parseSpecFields: scopes to Required/Optional, skips the map-key row", () => {
  const { requiredFields, optionalFields, mentioned } = parseSpecFields(SPEC);

  assert.deepEqual([...requiredFields].sort(), ["id"]);
  assert.deepEqual([...optionalFields].sort(), ["evidence", "label"]);

  // The "Entity ID (map key)" row is a structural note, never a field.
  assert.ok(!requiredFields.has("Entity ID (map key)"));
  // The nested-table row sits under "Subject Object", outside scope.
  assert.ok(!requiredFields.has("nested_only"));
  assert.ok(!optionalFields.has("nested_only"));
  assert.ok(!mentioned.has("nested_only"));

  // Combined-row backtick tokens are all "mentioned" even though only `evidence`
  // is its own field row.
  for (const tok of ["id", "label", "evidence", "citations", "sources", "media"]) {
    assert.ok(mentioned.has(tok), `expected ${tok} to be mentioned`);
  }
});

test("compareEntity: a fully aligned entity reports zero drift", () => {
  const schema = {
    additionalProperties: false,
    required: ["id"],
    properties: {
      id: {}, label: {}, evidence: {}, citations: {}, sources: {}, media: {},
    },
  };
  const r = compareEntity(SPEC, schema);
  assert.deepEqual(r.inSpecNotSchema, []);
  assert.deepEqual(r.inSchemaNotSpec, []);
  assert.deepEqual(r.specRequiredNotSchema, []);
  assert.deepEqual(r.specOptionalButSchemaRequired, []);
  assert.equal(r.apFalse, true);
});

test("compareEntity: a field documented in the spec but absent from the schema", () => {
  // Drop `label` from the schema — the spec documents it, the schema lacks it.
  const schema = {
    additionalProperties: false,
    required: ["id"],
    properties: { id: {}, evidence: {}, citations: {}, sources: {}, media: {} },
  };
  const r = compareEntity(SPEC, schema);
  assert.deepEqual(r.inSpecNotSchema, ["label"]);
  // additionalProperties:false is the dangerous case — surfaced to the caller.
  assert.equal(r.apFalse, true);
  assert.deepEqual(r.inSchemaNotSpec, []);
});

test("compareEntity: a schema field never documented in the spec tables", () => {
  const schema = {
    additionalProperties: false,
    required: ["id"],
    properties: {
      id: {}, label: {}, evidence: {}, citations: {}, sources: {}, media: {},
      orphan: {}, // present in schema, mentioned nowhere in the spec tables
    },
  };
  const r = compareEntity(SPEC, schema);
  assert.deepEqual(r.inSchemaNotSpec, ["orphan"]);
  assert.deepEqual(r.inSpecNotSchema, []);
});

test("compareEntity: combined-row tokens are not flagged as undocumented", () => {
  // citations/sources/media appear only inside the combined `evidence` row, not
  // as their own rows; they must still count as documented.
  const schema = {
    additionalProperties: false,
    required: ["id"],
    properties: { id: {}, label: {}, evidence: {}, citations: {}, sources: {}, media: {} },
  };
  const r = compareEntity(SPEC, schema);
  assert.deepEqual(r.inSchemaNotSpec, []);
});

test("compareEntity: required/optional mismatches in both directions", () => {
  // `id` is under Required Fields but the schema omits it from required[];
  // `label` is under Optional Fields but the schema marks it required.
  const schema = {
    additionalProperties: false,
    required: ["label"],
    properties: {
      id: {}, label: {}, evidence: {}, citations: {}, sources: {}, media: {},
    },
  };
  const r = compareEntity(SPEC, schema);
  assert.deepEqual(r.specRequiredNotSchema, ["id"]);
  assert.deepEqual(r.specOptionalButSchemaRequired, ["label"]);
  // Presence is unaffected — both fields exist in the schema.
  assert.deepEqual(r.inSpecNotSchema, []);
  assert.deepEqual(r.inSchemaNotSpec, []);
});

test("compareEntity: additionalProperties true is reported as apFalse=false", () => {
  const schema = {
    additionalProperties: true,
    required: ["id"],
    properties: { id: {}, label: {}, evidence: {}, citations: {}, sources: {}, media: {} },
  };
  const r = compareEntity(SPEC, schema);
  assert.equal(r.apFalse, false);
});

// --- scanTree: the filesystem-level pairing rules (#309, blocking) ----------
//
// These exercise run()'s scan against fixture trees, which is where the
// blocking gate can go silently green: a spec page with no schema, a schema
// with no spec page, a malformed schema, and a run that found no pairs at all.

const THING_SCHEMA = {
  additionalProperties: false,
  required: ["id"],
  properties: { id: {}, label: {}, evidence: {}, citations: {}, sources: {}, media: {} },
};

// Build a {specDir, schemaDir} pair under a fresh temp dir. `specs` and
// `schemas` map stem -> file contents (schema values are JSON.stringify'd
// unless already a string, so a test can write deliberately broken JSON).
const silentLog = { log() {}, warn() {}, error() {} };

function captureLog() {
  const lines = [];
  const write = (message) => lines.push(message);
  return { lines, log: write, warn: write, error: write };
}

function fixtureTree(t, specs, schemas) {
  const root = mkdtempSync(join(tmpdir(), "spec-schema-drift-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const specDir = join(root, "4-entity-types");
  const schemaDir = join(root, "schema", "v1");
  mkdirSync(specDir, { recursive: true });
  mkdirSync(schemaDir, { recursive: true });
  for (const [stem, body] of Object.entries(specs)) {
    writeFileSync(join(specDir, `${stem}.md`), body);
  }
  for (const [stem, body] of Object.entries(schemas)) {
    writeFileSync(
      join(schemaDir, `${stem}.schema.json`),
      typeof body === "string" ? body : JSON.stringify(body),
    );
  }
  return { specDir, schemaDir, log: silentLog };
}

test("scanTree: an aligned spec/schema pair reports zero drift", (t) => {
  const dirs = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA });
  assert.deepEqual(scanTree(dirs), { checked: 1, mismatches: 0 });
});

test("scanTree: a spec page with no schema is drift, not a silent skip", (t) => {
  const dirs = fixtureTree(t, { thing: SPEC, ghost: SPEC }, { thing: THING_SCHEMA });
  const r = scanTree(dirs);
  assert.equal(r.checked, 1);
  assert.equal(r.mismatches, 1);
});

test("scanTree: the allowlisted page is exempt only when its schema is absent", (t) => {
  // vocabularies.md has no schema by design — exempt.
  const absent = fixtureTree(t, { thing: SPEC, vocabularies: SPEC }, { thing: THING_SCHEMA });
  assert.deepEqual(scanTree(absent), { checked: 1, mismatches: 0 });

  // But a vocabularies.schema.json that exists and is broken is still drift:
  // the allowlist exempts "no schema", not "unreadable schema".
  const broken = fixtureTree(
    t,
    { thing: SPEC, vocabularies: SPEC },
    { thing: THING_SCHEMA, vocabularies: "{ not json" },
  );
  assert.equal(scanTree(broken).mismatches, 1);

  const unreadable = fixtureTree(t, { thing: SPEC, vocabularies: SPEC }, { thing: THING_SCHEMA });
  mkdirSync(join(unreadable.schemaDir, "vocabularies.schema.json"));
  assert.equal(scanTree(unreadable).mismatches, 1);
});

test("scanTree: a malformed schema is one drift finding", (t) => {
  const dirs = fixtureTree(t, { thing: SPEC }, { thing: "{ not json" });
  const log = captureLog();
  const r = scanTree({ ...dirs, log });
  assert.equal(r.checked, 0);
  assert.equal(r.mismatches, 1);
  assert.ok(log.lines.some((line) => line.includes("not valid JSON")));
  assert.ok(log.lines.some((line) => line.includes("1 drift finding(s)")));
  assert.ok(!log.lines.some((line) => line.includes("no entity spec/schema pairs")));
});

test("scanTree: a schema with no spec page is drift", (t) => {
  const dirs = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA, orphan: THING_SCHEMA });
  const r = scanTree(dirs);
  assert.equal(r.checked, 1);
  assert.equal(r.mismatches, 1);
});

test("scanTree: glx-file needs no spec page under 4-entity-types", (t) => {
  const dirs = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA, "glx-file": THING_SCHEMA });
  assert.deepEqual(scanTree(dirs), { checked: 1, mismatches: 0 });
});

test("scanTree: a run that compared nothing is drift, not a pass", (t) => {
  const dirs = fixtureTree(t, {}, {});
  assert.deepEqual(scanTree(dirs), { checked: 0, mismatches: 1 });
});

test("scanTree: missing directories report drift without throwing", (t) => {
  const missingSpec = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA });
  rmSync(missingSpec.specDir, { recursive: true });
  assert.deepEqual(scanTree(missingSpec), { checked: 0, mismatches: 1 });

  const missingSchema = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA });
  rmSync(missingSchema.schemaDir, { recursive: true });
  assert.deepEqual(scanTree(missingSchema), { checked: 0, mismatches: 1 });
});

test("scanTree: an unreadable spec page does not stop other comparisons", (t) => {
  const dirs = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA });
  mkdirSync(join(dirs.specDir, "broken.md"));
  assert.deepEqual(scanTree(dirs), { checked: 1, mismatches: 1 });
});

test("scanTree: invalid schema roots and fields report drift instead of crashing", (t) => {
  for (const body of ["null", "[]", "true", '{"required": 3}', '{"properties": []}']) {
    const dirs = fixtureTree(t, { thing: SPEC }, { thing: body });
    assert.deepEqual(scanTree(dirs), { checked: 0, mismatches: 1 }, body);
  }
});

test("run: strict mode fails and local mode reports for pairing and scan failures", (t) => {
  for (const [specs, schemas] of [
    [{ thing: SPEC, ghost: SPEC }, { thing: THING_SCHEMA }],
    [{ thing: SPEC }, { thing: "{ invalid json" }],
    [{}, {}],
  ]) {
    const dirs = fixtureTree(t, specs, schemas);
    assert.equal(run({ ...dirs, env: {} }), 0);
    assert.equal(run({ ...dirs, env: { DRIFT_STRICT: "1" } }), 1);
  }

  const missingSpec = fixtureTree(t, { thing: SPEC }, { thing: THING_SCHEMA });
  rmSync(missingSpec.specDir, { recursive: true });
  assert.equal(run({ ...missingSpec, env: {} }), 0);
  assert.equal(run({ ...missingSpec, env: { DRIFT_STRICT: "1" } }), 1);
});

test("exitCodeFor: drift fails only under DRIFT_STRICT=1", () => {
  assert.equal(exitCodeFor(0, {}), 0);
  assert.equal(exitCodeFor(0, { DRIFT_STRICT: "1" }), 0);
  // Warn-only default: a local run reports drift but still exits 0.
  assert.equal(exitCodeFor(3, {}), 0);
  assert.equal(exitCodeFor(3, { DRIFT_STRICT: "0" }), 0);
  assert.equal(exitCodeFor(3, { DRIFT_STRICT: "true" }), 0);
  // CI: blocking.
  assert.equal(exitCodeFor(1, { DRIFT_STRICT: "1" }), 1);
});
