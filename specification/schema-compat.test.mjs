// Unit tests for the schema<->schema backward-compatibility classifier (#311).
//
// classifySchemaChange is the verdict engine behind the hard-fail CI gate, so
// every branch is pinned: new / deleted / invalid-JSON (both sides) and the
// compatible-vs-breaking call delegated to json-schema-diff-validator (add
// optional vs. add-required / remove / tighten).
//
// Run from specification/ so json-schema-diff-validator resolves:
//   node --test schema-compat.test.mjs
// (or `node --test specification/schema-compat.test.mjs` from the repo root —
//  the library resolves relative to schema-compat.mjs, not the cwd).

import { test } from "node:test";
import assert from "node:assert/strict";
import { classifySchemaChange, normalizeDialect } from "./schema-compat.mjs";

const PATH = "specification/schema/v1/thing.schema.json";

// A small but realistic base schema with additionalProperties:false (the case
// where presence drift actually invalidates data).
const BASE = {
  $id: "https://genealogix.org/schema/v1/thing.schema.json",
  type: "object",
  additionalProperties: false,
  required: ["id"],
  properties: { id: { type: "string" }, name: { type: "string" } },
};
const json = (o) => JSON.stringify(o);

test("new schema: no base version is never breaking", () => {
  const r = classifySchemaChange({ path: PATH, baseContent: null, currentContent: json(BASE) });
  assert.equal(r.status, "new");
  assert.equal(r.breaking, false);
});

test("deleted schema: removing a v1 schema is breaking", () => {
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: null });
  assert.equal(r.status, "deleted");
  assert.equal(r.breaking, true);
  assert.match(r.message, /deleted/);
});

test("identical schema is backward compatible", () => {
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: json(BASE) });
  assert.equal(r.status, "compatible");
  assert.equal(r.breaking, false);
});

test("adding an optional property is backward compatible", () => {
  const next = { ...BASE, properties: { ...BASE.properties, note: { type: "string" } } };
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: json(next) });
  assert.equal(r.status, "compatible");
  assert.equal(r.breaking, false);
});

test("adding a new required property is breaking", () => {
  const next = {
    ...BASE,
    required: ["id", "name"],
    properties: { ...BASE.properties, extra: { type: "string" } },
  };
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: json(next) });
  assert.equal(r.status, "breaking");
  assert.equal(r.breaking, true);
  assert.match(r.message, /BREAKING/);
});

test("removing a property is breaking", () => {
  const next = {
    ...BASE,
    properties: { id: { type: "string" } }, // dropped `name`
  };
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: json(next) });
  assert.equal(r.status, "breaking");
  assert.equal(r.breaking, true);
});

test("tightening a property (adding a pattern) is breaking", () => {
  const next = {
    ...BASE,
    properties: { ...BASE.properties, id: { type: "string", pattern: "^x" } },
  };
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: json(next) });
  assert.equal(r.status, "breaking");
  assert.equal(r.breaking, true);
});

test("malformed base JSON is reported, not silently passed", () => {
  const r = classifySchemaChange({ path: PATH, baseContent: "{not json", currentContent: json(BASE) });
  assert.equal(r.status, "invalid-base");
  assert.equal(r.breaking, true);
});

test("malformed current JSON is reported, not silently passed", () => {
  const r = classifySchemaChange({ path: PATH, baseContent: json(BASE), currentContent: "{not json" });
  assert.equal(r.status, "invalid-current");
  assert.equal(r.breaking, true);
});

// --- 2020-12 dialect normalization (#794) ---
// The draft-07 → 2020-12 migration renames keywords without changing what
// validates; the gate must treat that rename as compatible while still
// catching real structural breaks expressed in either dialect.

const DRAFT07 = {
  $schema: "http://json-schema.org/draft-07/schema#",
  $id: "https://genealogix.org/schema/v1/thing.schema.json",
  type: "object",
  additionalProperties: false,
  required: ["id"],
  properties: {
    id: { type: "string" },
    entry: { $ref: "#/definitions/Entry" },
    lat: { type: "number" },
    lon: { type: "number" },
  },
  dependencies: { lat: ["lon"], lon: ["lat"] },
  definitions: { Entry: { type: "object", properties: { label: { type: "string" } } } },
};

// The same schema, mechanically migrated to 2020-12.
const MIGRATED = {
  ...DRAFT07,
  $schema: "https://json-schema.org/draft/2020-12/schema",
  properties: { ...DRAFT07.properties, entry: { $ref: "#/$defs/Entry" } },
  dependentRequired: DRAFT07.dependencies,
  $defs: DRAFT07.definitions,
};
delete MIGRATED.dependencies;
delete MIGRATED.definitions;

test("pure draft-07 → 2020-12 keyword migration is backward compatible", () => {
  const r = classifySchemaChange({ path: PATH, baseContent: json(DRAFT07), currentContent: json(MIGRATED) });
  assert.equal(r.status, "compatible");
  assert.equal(r.breaking, false);
});

test("removing a property during the dialect migration is still breaking", () => {
  const next = { ...MIGRATED, properties: { ...MIGRATED.properties } };
  delete next.properties.id;
  const r = classifySchemaChange({ path: PATH, baseContent: json(DRAFT07), currentContent: json(next) });
  assert.equal(r.status, "breaking");
  assert.equal(r.breaking, true);
});

test("adding a required field during the dialect migration is still breaking", () => {
  const next = { ...MIGRATED, required: ["id", "lat"] };
  const r = classifySchemaChange({ path: PATH, baseContent: json(DRAFT07), currentContent: json(next) });
  assert.equal(r.status, "breaking");
  assert.equal(r.breaking, true);
});

test("normalizeDialect leaves property NAMES that look like keywords alone", () => {
  const s = { type: "object", properties: { $defs: { type: "string" } } };
  assert.deepEqual(normalizeDialect(s), s);
});

test("normalizeDialect copies enum/const values verbatim", () => {
  const s = { type: "object", enum: [{ $defs: 1 }], const: { dependentRequired: true } };
  assert.deepEqual(normalizeDialect(s), s);
});
