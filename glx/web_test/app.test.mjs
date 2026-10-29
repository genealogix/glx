// SPDX-FileCopyrightText: 2026 Oracynth, Inc.
// SPDX-License-Identifier: Apache-2.0

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { createContext, runInContext } from "node:vm";

// Only the SVG element operations used by treeNode are needed. Running the
// actual viewer keeps its formatting and truncation behavior under test;
// registering its route listeners does not start the app.
class SvgElement {
  constructor(tag) {
    this.tag = tag;
    this.attributes = {};
    this.children = [];
    this.textContent = "";
  }

  setAttribute(name, value) { this.attributes[name] = String(value); }
  appendChild(child) { this.children.push(child); }
  addEventListener() {}
}

const viewer = createContext({
  document: { createElementNS: (_namespace, tag) => new SvgElement(tag) },
  window: { addEventListener() {} },
});
runInContext(readFileSync(new URL("../web/app.js", import.meta.url), "utf8"), viewer);
const treeNode = runInContext("treeNode", viewer);

function renderNode(properties) {
  return treeNode({ id: "person-test", name: "Test Person", _x: 16, _y: 16, ...properties }, 168, 46);
}

function dateLine(node) {
  return node.children.find((child) => child.attributes.class === "tn-years")?.textContent;
}

function tooltip(node) {
  return node.children.find((child) => child.tag === "title").textContent;
}

test("tree tooltip keeps both BCE range endpoints when the date line is truncated", () => {
  const lifespan = "100 BCE/50 BCE – 40 BCE/30 BCE";
  const node = renderNode({ lifespan, birthYear: -100, deathYear: -40 });
  assert.equal(dateLine(node), "100 BCE/50 BCE – 40 …");
  assert.equal(tooltip(node), `Test Person\n${lifespan}`);
});

test("tree tooltip keeps qualified lifespans instead of numeric fallback years", () => {
  const lifespan = "c. 560 BCE – bef. 500 BCE";
  const node = renderNode({ lifespan, birthYear: -560, deathYear: -500 });
  assert.ok(dateLine(node).endsWith("…"));
  assert.equal(tooltip(node), `Test Person\n${lifespan}`);
});

test("tree tooltip includes the numeric lifespan for older API responses", () => {
  const node = renderNode({ birthYear: 1850, deathYear: 1920 });
  assert.equal(dateLine(node), "1850–1920");
  assert.equal(tooltip(node), "Test Person\n1850–1920");
});

test("an unknown lifespan leaves a name-only tooltip", () => {
  const node = renderNode({ lifespan: "", birthYear: 0, deathYear: 0 });
  assert.equal(dateLine(node), undefined);
  assert.equal(tooltip(node), "Test Person");
});

test("the tooltip uses the ID when no name is present and keeps text literal", () => {
  const node = renderNode({ name: "", id: "person-<literal>", lifespan: "b. bef. 1850" });
  assert.equal(tooltip(node), "person-<literal>\nb. bef. 1850");
});
