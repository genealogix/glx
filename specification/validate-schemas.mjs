#!/usr/bin/env node
// Schema validation script — replaces abandoned ajv-cli with direct ajv usage.
// Validates all GLX JSON schemas against the meta-schema and compiles them.

import Ajv from "ajv";
import addFormats from "ajv-formats";
import yaml from "js-yaml";
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

const metaAjv = new Ajv({ strict: true, allErrors: true });
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

const compileAjv = new Ajv({ strict: true, allErrors: true });
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
const refAjv = new Ajv({ strict: true, allErrors: true });
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

// --- Step 4: Validate vocabulary .glx instance files against their schemas ---
// The standard vocabularies are the data the schemas describe; this catches
// drift between a vocabulary .glx and its schema deterministically (issue #839),
// reusing refAjv so $refs to other schemas resolve.
console.log("\nValidating vocabulary .glx files against their schemas...");

const VOCAB_GLX_DIR = join(ROOT, "specification/5-standard-vocabularies");

for (const glxName of readdirSync(VOCAB_GLX_DIR).filter((f) => f.endsWith(".glx"))) {
  const stem = basename(glxName, ".glx");
  let schema;
  try {
    schema = loadJSON(join(VOCAB_DIR, `${stem}.schema.json`));
  } catch (e) {
    if (e.code === "ENOENT") {
      console.error(`${glxName} INVALID: no matching schema at vocabularies/${stem}.schema.json`);
    } else {
      // A malformed schema file (JSON parse error) is a different failure than
      // a missing one — don't hide it behind "no matching schema".
      console.error(`${glxName} INVALID: schema vocabularies/${stem}.schema.json could not be read: ${formatError(e)}`);
    }
    errors++;
    continue;
  }
  const validate = refAjv.getSchema(schema.$id);
  if (!validate) {
    console.error(`${glxName} INVALID: schema ${stem}.schema.json ($id ${schema.$id}) is not registered`);
    errors++;
    continue;
  }
  let data;
  try {
    data = yaml.load(readFileSync(join(VOCAB_GLX_DIR, glxName), "utf8"));
  } catch (e) {
    console.error(`${glxName} INVALID: YAML parse failed: ${formatError(e)}`);
    errors++;
    continue;
  }
  if (validate(data)) {
    console.log(`${glxName} valid against ${stem}.schema.json`);
  } else {
    console.error(`${glxName} INVALID against ${stem}.schema.json:`);
    console.error(validate.errors);
    errors++;
  }
}

if (errors > 0) {
  console.error(`\n${errors} schema(s) failed validation.`);
  process.exit(1);
}
