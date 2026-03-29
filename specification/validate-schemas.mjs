#!/usr/bin/env node
// Schema validation script — replaces abandoned ajv-cli with direct ajv usage.
// Validates all GLX JSON schemas against the meta-schema and compiles them.

import Ajv from "ajv";
import addFormats from "ajv-formats";
import { readFileSync, readdirSync } from "fs";
import { join, basename, dirname } from "path";
import { fileURLToPath } from "url";

// Resolve paths relative to the repo root (script lives in specification/)
const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, "..");

const SCHEMA_DIR = join(ROOT, "specification/schema/v1");
const VOCAB_DIR = join(SCHEMA_DIR, "vocabularies");
const META_SCHEMA = join(ROOT, "specification/schema/meta/schema.schema.json");

function loadJSON(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function globSchemas(dir) {
  return readdirSync(dir)
    .filter((f) => f.endsWith(".schema.json"))
    .map((f) => join(dir, f));
}

function formatError(e) {
  return e instanceof Error ? e.message : String(e);
}

let errors = 0;

// --- Step 1: Validate schemas against meta-schema ---
console.log("Validating schemas against meta-schema...");

const metaAjv = new Ajv({ strict: "log", allErrors: true });
addFormats(metaAjv);
const metaSchema = loadJSON(META_SCHEMA);
const validateMeta = metaAjv.compile(metaSchema);

for (const file of [...globSchemas(SCHEMA_DIR), ...globSchemas(VOCAB_DIR)]) {
  const schema = loadJSON(file);
  const valid = validateMeta(schema);
  if (valid) {
    console.log(`${file} valid`);
  } else {
    console.error(`${file} INVALID:`);
    console.error(validateMeta.errors);
    errors++;
  }
}

// --- Step 2: Compile each schema individually ---
console.log("\nCompiling schemas...");

const compileAjv = new Ajv({ strict: "log", allErrors: true });
addFormats(compileAjv);

// Add meta-schema
compileAjv.compile(metaSchema);
console.log(`schema ${META_SCHEMA} is valid`);

// Compile individual schemas (non-glx-file)
const entitySchemas = globSchemas(SCHEMA_DIR).filter(
  (f) => basename(f) !== "glx-file.schema.json"
);
const vocabSchemas = globSchemas(VOCAB_DIR);

for (const file of [...entitySchemas, ...vocabSchemas]) {
  try {
    compileAjv.compile(loadJSON(file));
    console.log(`schema ${file} is valid`);
  } catch (e) {
    console.error(`schema ${file} FAILED: ${formatError(e)}`);
    errors++;
  }
}

// --- Step 3: Compile glx-file.schema.json with all references ---
const refAjv = new Ajv({ strict: "log", allErrors: true });
addFormats(refAjv);

// Add all entity and vocabulary schemas as references
for (const file of [...entitySchemas, ...vocabSchemas]) {
  refAjv.addSchema(loadJSON(file));
}

try {
  const glxFileSchema = join(SCHEMA_DIR, "glx-file.schema.json");
  refAjv.compile(loadJSON(glxFileSchema));
  console.log(`schema ${glxFileSchema} is valid`);
} catch (e) {
  console.error(`schema glx-file.schema.json FAILED: ${formatError(e)}`);
  errors++;
}

if (errors > 0) {
  console.error(`\n${errors} schema(s) failed validation.`);
  process.exit(1);
}
