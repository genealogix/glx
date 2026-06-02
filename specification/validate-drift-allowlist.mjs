#!/usr/bin/env node
// Runtime/dependency notes:
// - Expected to run with the repository's supported Node.js version (ESM-enabled).
// - External packages required by this script: `ajv`, `ajv-formats`, `js-yaml`.
// - These dependencies are provided by `specification/package.json` (install there).
// - Preferred invocation is via `make check-drift-allowlist`.
//
// Validate .claude/drift-allowlist.yaml against .claude/drift-allowlist.schema.json.
//
// This is the OFFLINE structural check: JSON-schema conformance (incl. the
// temporary-vs-permanent invariant encoded as a per-entry oneOf) plus a
// duplicate file+symbol guard. The closed-issue staleness gate — failing loud
// when a temporary entry's tracking_issue is CLOSED — is deferred to the
// deterministic drift checkers tracked by genealogix/glx#673 and #295, which
// need GitHub issue-state lookups this offline validator deliberately avoids.

import Ajv from "ajv";
import addFormats from "ajv-formats";
import { readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import yaml from "js-yaml";

// Resolve paths relative to the repo root (script lives in specification/).
const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, "..");
const ALLOWLIST = join(ROOT, ".claude/drift-allowlist.yaml");
const SCHEMA = join(ROOT, ".claude/drift-allowlist.schema.json");

let errors = 0;

const ajv = new Ajv({ strict: true, allErrors: true });
addFormats(ajv);
const validate = ajv.compile(JSON.parse(readFileSync(SCHEMA, "utf8")));

const entries = yaml.load(readFileSync(ALLOWLIST, "utf8"));

if (!validate(entries)) {
  console.error(`${ALLOWLIST} INVALID:`);
  console.error(validate.errors);
  errors += validate.errors?.length ?? 1;
} else {
  // Schema validation has confirmed `entries` is an array of well-formed objects.
  // Guard against the "allowlist becomes a dumping ground" risk (genealogix/glx#797):
  // no two entries may suppress the same file+symbol. Key on the JSON-encoded
  // [file, symbol] tuple so values containing a separator can't collide.
  const seen = new Set();
  for (const e of entries) {
    const fileKey =
      typeof e.file === "string" ? ["string", e.file] : ["non-string", typeof e.file, String(e.file)];
    const symbolKey =
      typeof e.symbol === "string"
        ? ["string", e.symbol]
        : ["non-string", typeof e.symbol, String(e.symbol)];
    const key = JSON.stringify([fileKey, symbolKey]);
    if (seen.has(key)) {
      console.error(`${ALLOWLIST}: duplicate entry for ${e.file} :: ${e.symbol}`);
      errors++;
    }
    seen.add(key);
  }
}

if (errors > 0) {
  console.error(`\n${errors} drift-allowlist error(s).`);
  process.exit(1);
}

// Past the error gate above, schema validation has guaranteed `entries` is an array.
const count = entries.length;
console.log(`${ALLOWLIST} valid (${count} entr${count === 1 ? "y" : "ies"}).`);
