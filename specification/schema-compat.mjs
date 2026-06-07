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
//
// classifySchemaChange is the testable core: it takes the base and current
// schema text (no git, no filesystem) and returns the verdict, so the new/
// deleted/breaking/compatible branches are unit-tested in schema-compat.test.mjs.
// main() is the thin git/console wrapper, run only when invoked as a script.

import { readFileSync } from "fs";
import { join } from "path";
import { execFileSync } from "child_process";
import { createRequire } from "module";
import { pathToFileURL } from "url";

const require = createRequire(import.meta.url);
const { validateSchemaCompatibility } = require("json-schema-diff-validator");

// classifySchemaChange decides whether one schema's change is backward-compatible
// from the base text and current text alone (each a string, or null to signal
// "absent": baseContent === null means the schema is new; currentContent === null
// means it was deleted). Returns { path, status, breaking, message } where status
// is one of new | deleted | invalid-base | invalid-current | compatible | breaking.
export function classifySchemaChange({ path, baseContent, currentContent }) {
  // New schema: there is no base version, so nothing can break.
  if (baseContent == null) {
    return {
      path, status: "new", breaking: false,
      message: `+ ${path}: new schema (no base version) — nothing to break`,
    };
  }
  // Deleted schema: removing a v1 schema makes every archive that referenced it
  // invalid — a backward-incompatible change.
  if (currentContent == null) {
    return {
      path, status: "deleted", breaking: true,
      message: `✗ ${path}: schema deleted — removing a v1 schema is breaking (bump the schema version directory instead)`,
    };
  }
  let oldSchema;
  try {
    oldSchema = JSON.parse(baseContent);
  } catch (e) {
    return {
      path, status: "invalid-base", breaking: true,
      message: `✗ ${path}: base version is not valid JSON (${e.message})`,
    };
  }
  let newSchema;
  try {
    newSchema = JSON.parse(currentContent);
  } catch (e) {
    return {
      path, status: "invalid-current", breaking: true,
      message: `✗ ${path}: new version is not valid JSON (${e.message})`,
    };
  }
  try {
    validateSchemaCompatibility(oldSchema, newSchema);
    return { path, status: "compatible", breaking: false, message: `✓ ${path}: backward compatible` };
  } catch (e) {
    return { path, status: "breaking", breaking: true, message: `✗ ${path}: BREAKING change — ${e.message}` };
  }
}

// Run git with an argument array (no shell) so a crafted ref or filename in a
// PR cannot inject commands.
function git(args) {
  return execFileSync("git", args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
}

function main() {
  // Base ref: the PR base branch in CI, else origin/main locally.
  const base = process.env.GITHUB_BASE_REF
    ? `origin/${process.env.GITHUB_BASE_REF}`
    : "origin/main";

  // git emits repo-root-relative paths; resolve reads against the repo root so the
  // script works regardless of the directory it is invoked from (e.g. specification/).
  const repoRoot = git(["rev-parse", "--show-toplevel"]).trim();

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
    let baseContent = null;
    try {
      baseContent = git(["show", `${base}:${path}`]);
    } catch {
      baseContent = null; // not in the base ref → new schema
    }
    let currentContent = null;
    if (baseContent != null) {
      try {
        currentContent = readFileSync(join(repoRoot, path), "utf8");
      } catch (e) {
        if (e.code === "ENOENT") {
          currentContent = null; // deleted in this PR
        } else {
          // A non-ENOENT read error (permissions, etc.) is an environment fault,
          // not a schema verdict — report and count it breaking, but don't
          // misclassify it as a deletion.
          console.error(`✗ ${path}: could not read schema (${e.message})`);
          breaking++;
          continue;
        }
      }
    }

    const result = classifySchemaChange({ path, baseContent, currentContent });
    if (result.breaking) {
      console.error(result.message);
      breaking++;
    } else {
      console.log(result.message);
    }
    // checked counts only schemas that reached the actual compatibility check
    // (both versions present and valid JSON).
    if (result.status === "compatible" || result.status === "breaking") {
      checked++;
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
}

// Run the git diff only when invoked as a script, not when imported by the test
// file (which calls the exported classifySchemaChange against fixtures).
if (import.meta.url === pathToFileURL(process.argv[1] || "").href) {
  main();
}
