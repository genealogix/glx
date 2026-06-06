#!/usr/bin/env node
// Schema <-> schema backward-compatibility check (issue #311).
//
// For each schema changed in this PR, diff it against the base branch and fail
// if the change is backward-incompatible — i.e. it would make previously-valid
// archives invalid (removing a property under additionalProperties:false,
// tightening a pattern, adding a new required field, narrowing a type).
//
// HARD-FAIL: unlike the spec<->schema parity warn (#309), this blocks, because
// its whole job is stopping data-breaking schema merges. Lives in specification/
// so it resolves json-schema-diff-validator from specification/node_modules.

import { readFileSync } from "fs";
import { execFileSync } from "child_process";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { validateSchemaCompatibility } = require("json-schema-diff-validator");

// Base ref: the PR base branch in CI, else origin/main locally.
const base = process.env.GITHUB_BASE_REF
  ? `origin/${process.env.GITHUB_BASE_REF}`
  : "origin/main";

// Run git with an argument array (no shell) so a crafted ref or filename in a
// PR cannot inject commands.
function git(args) {
  return execFileSync("git", args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
}

let changed = [];
try {
  changed = git(["diff", "--name-only", `${base}...HEAD`, "--", "specification/schema/v1"])
    .split("\n")
    .map((s) => s.trim())
    .filter((f) => f.endsWith(".schema.json"));
} catch (e) {
  console.error(`Could not diff against ${base} (is it fetched? use fetch-depth: 0): ${e.message}`);
  process.exit(2);
}

let breaking = 0;
let checked = 0;

for (const path of changed) {
  let baseContent;
  try {
    baseContent = git(["show", `${base}:${path}`]);
  } catch {
    console.log(`+ ${path}: new schema (no base version) — nothing to break`);
    continue;
  }
  let oldSchema, newSchema;
  try {
    oldSchema = JSON.parse(baseContent);
  } catch (e) {
    console.error(`✗ ${path}: base version is not valid JSON (${e.message})`);
    breaking++;
    continue;
  }
  let newContent;
  try {
    newContent = readFileSync(path, "utf8");
  } catch (e) {
    if (e.code === "ENOENT") {
      // The schema was deleted in this PR. Removing a v1 schema makes every
      // archive that referenced it invalid — a backward-incompatible change.
      console.error(`✗ ${path}: schema deleted — removing a v1 schema is breaking (bump the schema version directory instead)`);
    } else {
      console.error(`✗ ${path}: could not read schema (${e.message})`);
    }
    breaking++;
    continue;
  }
  try {
    newSchema = JSON.parse(newContent);
  } catch (e) {
    console.error(`✗ ${path}: new version is not valid JSON (${e.message})`);
    breaking++;
    continue;
  }
  checked++;
  try {
    validateSchemaCompatibility(oldSchema, newSchema);
    console.log(`✓ ${path}: backward compatible`);
  } catch (e) {
    console.error(`✗ ${path}: BREAKING change — ${e.message}`);
    breaking++;
  }
}

console.log(
  `\nschema↔schema compatibility: ${checked} changed schema(s) checked vs ${base}, ${breaking} breaking.`,
);

if (breaking > 0) {
  console.error(
    "\nBreaking schema changes make existing valid archives invalid. If this is an " +
      "intentional major-version change, bump the schema version directory rather than " +
      "editing v1 in place.",
  );
  process.exit(1);
}
