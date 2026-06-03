// Copyright 2025 Oracynth, Inc.
// Licensed under the Apache License, Version 2.0.
//
// Read-only single-page viewer for `glx serve` (issue #93, Phase 1).
// Vanilla JS, no build step, no external dependencies. Hash-based routing so
// the Go file server only ever serves the static shell; data comes from /api.

"use strict";

const SVG_NS = "http://www.w3.org/2000/svg";
const app = () => document.getElementById("app");

// ---------------------------------------------------------------------------
// Tiny DOM helpers (build with createElement so archive data is never injected
// as HTML — text always goes through textContent).
// ---------------------------------------------------------------------------

function h(tag, attrs, ...children) {
  const el = document.createElement(tag);
  if (attrs) {
    for (const [k, v] of Object.entries(attrs)) {
      if (v == null || v === false) continue;
      if (k === "class") el.className = v;
      else if (k.startsWith("on") && typeof v === "function") el.addEventListener(k.slice(2), v);
      else el.setAttribute(k, v);
    }
  }
  appendChildren(el, children);
  return el;
}

function appendChildren(el, children) {
  for (const child of children.flat()) {
    if (child == null || child === false) continue;
    // Element.append() inserts strings/numbers as plain text (never parsed as
    // HTML) and appends DOM nodes as-is. No value passed here can become
    // markup, so user/URL/exception text reaching the DOM is always escaped.
    el.append(child);
  }
}

// safeHref returns url only if it uses a safe, non-script scheme. Archive data
// (e.g. imported from GEDCOM) can contain arbitrary URLs, so reject anything
// that isn't http(s)/mailto to avoid javascript: and data: URL injection.
function safeHref(url) {
  return typeof url === "string" && /^(https?:|mailto:)/i.test(url.trim()) ? url : null;
}

// externalLink builds a new-tab anchor for an archive-supplied URL, or returns
// null when the URL is missing or uses an unsafe scheme.
function externalLink(url, label) {
  const href = safeHref(url);
  return href ? h("a", { href, target: "_blank", rel: "noopener noreferrer" }, label || href) : null;
}

function render(node) {
  const root = app();
  root.replaceChildren(node);
}

function showError(message) {
  render(h("div", { class: "error" }, message));
}

async function api(path) {
  const res = await fetch(path, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      if (body && body.error) detail = body.error;
    } catch (_) { /* keep statusText */ }
    throw new Error(detail);
  }
  return res.json();
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

function lifespan(birth, death) {
  if (birth && death) return `${birth}–${death}`;
  if (birth) return `b. ${birth}`;
  if (death) return `d. ${death}`;
  return "";
}

function sexClass(sex) {
  if (sex === "male") return "sex-male";
  if (sex === "female") return "sex-female";
  return "sex-unknown";
}

function personLink(ref) {
  const span = h("a", { href: `#/person/${encodeURIComponent(ref.id)}` }, ref.name || ref.id);
  const years = lifespan(ref.birthYear, ref.deathYear);
  if (!years && !ref.detail) return span;
  return h("span", null, span, " ",
    h("span", { class: "ref-detail" }, ref.detail ? ref.detail : `(${years})`));
}

function setActiveNav(name) {
  document.querySelectorAll(".nav a").forEach((a) => {
    a.classList.toggle("active", a.getAttribute("data-nav") === name);
  });
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

const COUNT_LABELS = {
  persons: "People", events: "Events", relationships: "Relationships",
  places: "Places", sources: "Sources", citations: "Citations",
  repositories: "Repositories", assertions: "Assertions", media: "Media",
};

async function viewDashboard() {
  setActiveNav("dashboard");
  const data = await api("/api/overview");

  const cards = h("div", { class: "cards" },
    Object.keys(COUNT_LABELS).map((key) =>
      h("div", { class: "card" },
        h("div", { class: "num" }, data.counts[key] ?? 0),
        h("div", { class: "label" }, COUNT_LABELS[key]))));

  const sections = [
    h("h1", null, "Archive overview"),
    h("p", { class: "subtitle" }, data.archivePath),
    cards,
  ];

  if (data.confidence && data.confidence.length) {
    sections.push(h("h2", null, "Assertion confidence"));
    sections.push(h("div", { class: "panel" },
      data.confidence.map((row) => barRow(row.level, row.count, row.percent))));
  }

  if (data.coverage && data.coverage.length) {
    sections.push(h("h2", null, "Evidence coverage"));
    sections.push(h("div", { class: "panel" },
      data.coverage.map((row) =>
        barRow(row.label, `${row.covered}/${row.total}`, row.percent))));
  }

  render(h("div", null, sections));
}

function barRow(label, value, percent) {
  return h("div", { class: "bar-row" },
    h("div", null, label),
    h("div", { class: "bar-track" },
      h("div", { class: "bar-fill", style: `width:${Math.max(0, Math.min(100, percent)).toFixed(1)}%` })),
    h("div", { class: "bar-num" }, `${value} · ${percent.toFixed(0)}%`));
}

// ---------------------------------------------------------------------------
// People list
// ---------------------------------------------------------------------------

async function viewPersons() {
  setActiveNav("persons");
  const data = await api("/api/persons");
  const persons = data.persons || [];

  const tbody = h("tbody");
  const rows = persons.map((p) => {
    const tr = h("tr", { onclick: () => navigate(`#/person/${encodeURIComponent(p.id)}`) },
      h("td", null, p.name),
      h("td", { class: "years" }, lifespan(p.birthYear, p.deathYear)),
      h("td", { class: "muted" }, p.sex || ""));
    tr.dataset.name = (p.name || "").toLowerCase();
    return tr;
  });
  appendChildren(tbody, rows);

  const search = h("input", {
    class: "search",
    type: "search",
    placeholder: `Filter ${persons.length} people…`,
    oninput: (e) => {
      const q = e.target.value.toLowerCase();
      rows.forEach((tr) => {
        tr.style.display = tr.dataset.name.includes(q) ? "" : "none";
      });
    },
  });

  const table = h("table", null,
    h("thead", null, h("tr", null,
      h("th", null, "Name"), h("th", null, "Lifespan"), h("th", null, "Sex"))),
    tbody);

  render(h("div", null,
    h("h1", null, "People"),
    persons.length ? search : null,
    persons.length ? table : h("p", { class: "muted" }, "No people in this archive.")));
}

// ---------------------------------------------------------------------------
// Person detail
// ---------------------------------------------------------------------------

async function viewPerson(id) {
  setActiveNav("persons");
  const p = await api(`/api/persons/${encodeURIComponent(id)}`);

  const badges = h("div", { class: "badges" });
  if (p.sex) badges.appendChild(h("span", { class: `badge ${sexClass(p.sex)}` }, p.sex));
  if (p.gender && p.gender !== p.sex) badges.appendChild(h("span", { class: "badge" }, p.gender));
  if (p.living) badges.appendChild(h("span", { class: "badge living" }, "living"));

  const altNames = (p.names || []).slice(1);

  const sections = [
    h("div", { class: "crumbs" }, h("a", { href: "#/persons" }, "People"), " / ", p.names[0] || id),
    h("h1", null, p.names[0] || id),
    altNames.length ? h("p", { class: "subtitle" }, "Also: " + altNames.join(" · ")) : null,
    badges,
    h("div", { class: "toolbar" },
      h("a", { class: "btn", href: `#/tree/${encodeURIComponent(id)}` }, "🌳 View family tree")),
  ];

  if (p.vitals && p.vitals.length) {
    sections.push(h("h2", null, "Vital events"));
    sections.push(h("div", { class: "panel" }, h("ul", { class: "timeline" }, p.vitals.map(eventRow))));
  }

  sections.push(familySection(p.family));

  if (p.events && p.events.length) {
    sections.push(h("h2", null, `Timeline (${p.events.length})`));
    sections.push(h("div", { class: "panel" }, h("ul", { class: "timeline" }, p.events.map(eventRow))));
  }

  if (p.assertions && p.assertions.length) {
    sections.push(h("h2", null, `Assertions (${p.assertions.length})`));
    sections.push(h("div", { class: "panel" }, p.assertions.map(assertionRow)));
  }

  if (p.notes && p.notes.length) {
    sections.push(h("h2", null, "Notes"));
    sections.push(h("div", { class: "panel" }, p.notes.map((n) => h("p", null, n))));
  }

  render(h("div", null, sections));
}

function eventRow(ev) {
  const place = ev.place ? ` — ${ev.place}` : "";
  return h("li", null,
    h("span", { class: "date" }, ev.date || "—"),
    h("span", null,
      h("span", { class: "etype" }, ev.typeLabel || ev.type),
      place,
      ev.role && ev.role !== "principal" && ev.role !== "subject"
        ? h("span", { class: "role" }, ` (${ev.role})`) : null));
}

function familySection(family) {
  if (!family) return null;
  const cols = [
    familyColumn("Parents", family.parents),
    familyColumn("Spouses", family.spouses),
    familyColumn("Children", family.children),
    familyColumn("Siblings", family.siblings),
  ].filter(Boolean);
  if (!cols.length) return null;
  return h("div", null,
    h("h2", null, "Family"),
    h("div", { class: "panel" }, h("div", { class: "family-grid" }, cols)));
}

function familyColumn(title, refs) {
  if (!refs || !refs.length) return null;
  return h("div", { class: "family-col" },
    h("h3", null, `${title} (${refs.length})`),
    h("ul", null, refs.map((r) => h("li", null, personLink(r)))));
}

function assertionRow(a) {
  const claim = a.property
    ? `${prettyProperty(a.property)}: ${a.value || "—"}`
    : (a.value || "(claim)");
  const head = h("div", { class: "claim" }, claim);
  if (a.confidence) {
    head.appendChild(h("span", { class: `conf conf-${a.confidence}` }, a.confidence));
  }
  const parts = [head];
  if (a.date) parts.push(h("div", { class: "meta" }, a.date));
  if (a.evidence && a.evidence.length) {
    parts.push(h("ul", { class: "evidence" }, a.evidence.map(evidenceRow)));
  } else {
    parts.push(h("div", { class: "meta" }, "⚠ no evidence cited"));
  }
  return h("div", { class: "assertion" }, parts);
}

function evidenceRow(ev) {
  const label = ev.sourceTitle || ev.sourceId || ev.citationId || "source";
  const text = ev.locator ? `${label} — ${ev.locator}` : label;
  const link = ev.sourceId
    ? h("a", { href: `#/source/${encodeURIComponent(ev.sourceId)}` }, text)
    : document.createTextNode(text);
  const li = h("li", null, link);
  const url = safeHref(ev.url);
  if (url) {
    li.appendChild(document.createTextNode(" "));
    li.appendChild(h("a", { href: url, target: "_blank", rel: "noopener noreferrer" }, "↗"));
  }
  return li;
}

function prettyProperty(p) {
  return p.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

// ---------------------------------------------------------------------------
// Family tree (SVG pedigree / descendancy)
// ---------------------------------------------------------------------------

const TREE_DIR_KEY = "glx.tree.dir";
const TREE_GEN_KEY = "glx.tree.gen";

async function viewTree(id) {
  setActiveNav("persons");
  const dir = sessionStorage.getItem(TREE_DIR_KEY) || "ancestors";
  const gen = parseInt(sessionStorage.getItem(TREE_GEN_KEY) || "4", 10);
  const data = await api(`/api/persons/${encodeURIComponent(id)}/tree?dir=${dir}&gen=${gen}`);

  const dirButton = (value, label) =>
    h("button", {
      class: "btn" + (dir === value ? " active" : ""),
      onclick: () => { sessionStorage.setItem(TREE_DIR_KEY, value); viewTree(id).catch((e) => showError(e.message)); },
    }, label);

  const genSelect = h("select", {
    onchange: (e) => { sessionStorage.setItem(TREE_GEN_KEY, e.target.value); viewTree(id).catch((err) => showError(err.message)); },
  }, [2, 3, 4, 5, 6, 8, 10].map((n) =>
    h("option", { value: n, selected: n === gen ? "selected" : null }, `${n} generations`)));

  const toolbar = h("div", { class: "toolbar" },
    h("a", { class: "btn", href: `#/person/${encodeURIComponent(id)}` }, "← Profile"),
    dirButton("ancestors", "Ancestors"),
    dirButton("descendants", "Descendants"),
    genSelect);

  render(h("div", null,
    h("h1", null, data.root.name || id),
    h("p", { class: "subtitle" }, dir === "ancestors" ? "Pedigree chart" : "Descendancy chart"),
    toolbar,
    h("div", { class: "tree-wrap" }, buildTreeSVG(data.root))));
}

// buildTreeSVG lays out a nested tree left-to-right: depth → x column, and each
// node centered on its children. Leaves stack vertically.
function buildTreeSVG(root) {
  const COL_W = 200, ROW_H = 64, BOX_W = 168, BOX_H = 46, PAD = 16;

  let leaf = 0;
  let maxDepth = 0;
  (function layout(node, depth) {
    node._x = depth * COL_W + PAD;
    maxDepth = Math.max(maxDepth, depth);
    if (node.children && node.children.length) {
      node.children.forEach((c) => layout(c, depth + 1));
      const first = node.children[0]._y;
      const last = node.children[node.children.length - 1]._y;
      node._y = (first + last) / 2;
    } else {
      node._y = leaf * ROW_H + PAD;
      leaf += 1;
    }
  })(root, 0);

  const width = (maxDepth + 1) * COL_W + PAD;
  const height = Math.max(leaf, 1) * ROW_H + PAD;

  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("width", width);
  svg.setAttribute("height", height);
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);

  const edges = document.createElementNS(SVG_NS, "g");
  const nodes = document.createElementNS(SVG_NS, "g");
  svg.appendChild(edges);
  svg.appendChild(nodes);

  (function draw(node) {
    if (node.children) {
      for (const child of node.children) {
        edges.appendChild(elbow(
          node._x + BOX_W, node._y + BOX_H / 2,
          child._x, child._y + BOX_H / 2));
        draw(child);
      }
    }
    nodes.appendChild(treeNode(node, BOX_W, BOX_H));
  })(root);

  return svg;
}

function elbow(x1, y1, x2, y2) {
  const midX = (x1 + x2) / 2;
  const path = document.createElementNS(SVG_NS, "path");
  path.setAttribute("class", "tree-edge");
  path.setAttribute("d", `M${x1},${y1} H${midX} V${y2} H${x2}`);
  return path;
}

function treeNode(node, w, hh) {
  const g = document.createElementNS(SVG_NS, "g");
  g.setAttribute("class", `tree-node ${sexClass(node.sex)}`);
  g.setAttribute("transform", `translate(${node._x},${node._y})`);
  g.addEventListener("click", () => navigate(`#/person/${encodeURIComponent(node.id)}`));

  const rect = document.createElementNS(SVG_NS, "rect");
  rect.setAttribute("width", w);
  rect.setAttribute("height", hh);
  rect.setAttribute("rx", "6");
  g.appendChild(rect);

  const bar = document.createElementNS(SVG_NS, "line");
  bar.setAttribute("class", `tn-bar tn-bar-${node.sex || "unknown"}`);
  bar.setAttribute("x1", "2"); bar.setAttribute("y1", "4");
  bar.setAttribute("x2", "2"); bar.setAttribute("y2", String(hh - 4));
  g.appendChild(bar);

  g.appendChild(svgText(node.name || node.id, 12, 19, "tn-name", w - 20));
  const years = lifespan(node.birthYear, node.deathYear);
  if (years) g.appendChild(svgText(years, 12, 35, "tn-years", w - 20));

  const title = document.createElementNS(SVG_NS, "title");
  title.textContent = node.name || node.id;
  g.appendChild(title);

  return g;
}

function svgText(text, x, y, cls, maxWidth) {
  const t = document.createElementNS(SVG_NS, "text");
  t.setAttribute("x", x);
  t.setAttribute("y", y);
  t.setAttribute("class", cls);
  // Rough truncation so long names don't overflow the box.
  const max = Math.floor(maxWidth / 7);
  t.textContent = text.length > max ? text.slice(0, max - 1) + "…" : text;
  return t;
}

// ---------------------------------------------------------------------------
// Sources
// ---------------------------------------------------------------------------

async function viewSources() {
  setActiveNav("sources");
  const data = await api("/api/sources");
  const sources = data.sources || [];

  const tbody = h("tbody");
  appendChildren(tbody, sources.map((s) =>
    h("tr", { onclick: () => navigate(`#/source/${encodeURIComponent(s.id)}`) },
      h("td", null, s.title),
      h("td", { class: "muted" }, s.type || ""),
      h("td", { class: "years" }, s.date || ""),
      h("td", { class: "years" }, String(s.citationCount)))));

  const table = h("table", null,
    h("thead", null, h("tr", null,
      h("th", null, "Title"), h("th", null, "Type"),
      h("th", null, "Date"), h("th", null, "Citations"))),
    tbody);

  render(h("div", null,
    h("h1", null, `Sources (${sources.length})`),
    sources.length ? table : h("p", { class: "muted" }, "No sources in this archive.")));
}

async function viewSource(id) {
  setActiveNav("sources");
  const s = await api(`/api/sources/${encodeURIComponent(id)}`);

  const meta = [];
  if (s.type) meta.push(`Type: ${s.type}`);
  if (s.authors && s.authors.length) meta.push(`By ${s.authors.join(", ")}`);
  if (s.date) meta.push(s.date);
  if (s.repository) meta.push(`Repository: ${s.repository}`);

  const sections = [
    h("div", { class: "crumbs" }, h("a", { href: "#/sources" }, "Sources"), " / ", s.title),
    h("h1", null, s.title),
    meta.length ? h("p", { class: "subtitle" }, meta.join(" · ")) : null,
    safeHref(s.url) ? h("p", null, externalLink(s.url, s.url)) : null,
  ];

  if (s.notes && s.notes.length) {
    sections.push(h("div", { class: "panel" }, s.notes.map((n) => h("p", null, n))));
  }

  sections.push(h("h2", null, `Citations (${s.citations.length})`));
  if (s.citations.length) {
    sections.push(h("div", { class: "panel" }, s.citations.map(citationRow)));
  } else {
    sections.push(h("p", { class: "muted" }, "No citations reference this source."));
  }

  render(h("div", null, sections));
}

function citationRow(c) {
  const parts = [];
  if (c.locator) parts.push(h("div", { class: "claim" }, c.locator));
  if (c.text) parts.push(h("div", null, "“", c.text, "”"));
  const sub = [];
  if (c.accessed) sub.push(`accessed ${c.accessed}`);
  if (sub.length) parts.push(h("div", { class: "meta" }, sub.join(" · ")));
  const cUrl = externalLink(c.url, c.url);
  if (cUrl) parts.push(h("div", null, cUrl));
  if (!parts.length) parts.push(h("div", { class: "meta" }, c.id));
  return h("div", { class: "assertion" }, parts);
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

function navigate(hash) {
  if (location.hash === hash) route();
  else location.hash = hash;
}

function route() {
  const hash = location.hash || "#/";
  const parts = hash.slice(2).split("/"); // strip "#/"
  const [section, rawId] = parts;
  const id = rawId ? decodeURIComponent(rawId) : "";

  app().replaceChildren(h("p", { class: "muted" }, "Loading…"));

  let view;
  switch (section) {
    case "": view = viewDashboard(); break;
    case "persons": view = viewPersons(); break;
    case "person": view = viewPerson(id); break;
    case "tree": view = viewTree(id); break;
    case "sources": view = viewSources(); break;
    case "source": view = viewSource(id); break;
    default: view = Promise.reject(new Error("Page not found: " + hash));
  }
  view.catch((err) => showError(err.message || String(err)));
  window.scrollTo(0, 0);
}

window.addEventListener("hashchange", route);
window.addEventListener("DOMContentLoaded", route);
