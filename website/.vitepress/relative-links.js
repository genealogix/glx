// Resolve relative markdown links against their *source* file and map them
// through the `rewrites` table, so one link form works on GitHub and here.
//
// VitePress renders a relative link relative to the page's rewritten URL, not
// its source path. With `docs/quickstart.md` served at `/quickstart` and
// `CONTRIBUTING.md` at `/development/contributing`, a link such as
// `[x](CONTRIBUTING.md)` from the README lands on `/CONTRIBUTING`, which does
// not exist, and the built-in dead-link check resolves against the source path
// so it never notices. The docs had worked around this with absolute routes
// (`/quickstart`), which 404 on GitHub (#1196).
//
// This hook runs before VitePress's own link handling: for every relative link
// whose target is a markdown page under srcDir it substitutes the absolute
// rewritten route, and VitePress then normalises that (`.md` -> `.html`, base
// prefix) and validates it as usual. Links that are external, absolute,
// anchors, or point at non-markdown files are left untouched.

import fs from 'node:fs'
import path from 'node:path'

/**
 * Build a `page -> rewritten page` function with the same first-match
 * semantics VitePress applies to the `rewrites` object. Supports the two
 * key shapes used in config.js: exact paths and `prefix/:page*` wildcards.
 */
export function createRewriteResolver(rewrites) {
  const rules = Object.entries(rewrites).map(([from, to]) => {
    const wildcard = from.indexOf('/:')
    if (wildcard === -1) {
      if (from.includes(':') || from.startsWith('^')) {
        throw new Error(`relative-links: unsupported rewrite key ${JSON.stringify(from)}`)
      }
      return { exact: from, to }
    }
    if (!from.endsWith('/:page*') || !to.endsWith('/:page*')) {
      throw new Error(`relative-links: unsupported rewrite rule ${JSON.stringify(from)} -> ${JSON.stringify(to)}`)
    }
    return { prefix: from.slice(0, wildcard + 1), toPrefix: to.slice(0, to.indexOf('/:') + 1) }
  })
  return (page) => {
    for (const rule of rules) {
      if (rule.exact !== undefined) {
        if (rule.exact === page) return rule.to
      } else if (page.startsWith(rule.prefix)) {
        return rule.toPrefix + page.slice(rule.prefix.length)
      }
    }
    return page
  }
}

/**
 * Map one href to its absolute rewritten route, or return null to leave it
 * alone. `fromPage` is the source page path relative to srcDir
 * (e.g. `docs/examples/README.md`).
 */
export function resolveRelativeMarkdownLink(href, fromPage, srcDir, rewrite) {
  if (/^([a-z][a-z0-9+.-]*:|\/|#)/i.test(href)) return null
  const [, target, suffix = ''] = href.match(/^([^?#]*)([?#].*)?$/)
  if (!target) return null

  let page = path.posix.normalize(path.posix.join(path.posix.dirname(fromPage), target))
  if (page === '..' || page.startsWith('../')) return null

  const abs = path.join(srcDir, page)
  let stat
  try {
    stat = fs.statSync(abs)
  } catch {
    return null
  }
  if (stat.isDirectory()) {
    // GitHub renders README.md for a directory link; VitePress serves index.md.
    const index = ['README.md', 'index.md'].find((name) => fs.existsSync(path.join(abs, name)))
    if (!index) return null
    page = path.posix.join(page, index)
  } else if (!page.endsWith('.md')) {
    return null
  }
  return '/' + rewrite(page) + suffix
}

/** markdown-it plugin: `md.use(relativeLinksPlugin, { srcDir, rewrites })`. */
export function relativeLinksPlugin(md, { srcDir, rewrites }) {
  const rewrite = createRewriteResolver(rewrites)
  const render = md.renderer.rules.link_open
  md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
    const token = tokens[idx]
    const hrefIndex = token.attrIndex('href')
    // env.relativePath is the *rewritten* page path (e.g. `cli.md` for
    // docs/cli/index.md); env.realPath is the file on disk, which is what the
    // author's relative link is relative to.
    const sourceFile = env?.realPath || env?.path
    if (hrefIndex >= 0 && sourceFile) {
      const fromPage = path.relative(srcDir, sourceFile).split(path.sep).join('/')
      const mapped = resolveRelativeMarkdownLink(token.attrs[hrefIndex][1], fromPage, srcDir, rewrite)
      if (mapped) token.attrs[hrefIndex][1] = mapped
    }
    return render ? render(tokens, idx, options, env, self) : self.renderToken(tokens, idx, options)
  }
}
