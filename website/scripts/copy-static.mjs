// Copies non-markdown reference files (JSON schemas) from `specification/` into
// the VitePress `public/` directory so that relative links like
// `../schema/v1/person.schema.json` from the rendered entity spec pages resolve
// to a real file on the published site.
//
// VitePress only copies .md → .html during the build; everything else under
// srcDir is ignored. We could symlink the trees, but a copy is simpler and
// works the same on every platform / CI runner.
//
// Run automatically by `npm run dev` and `npm run build` (see package.json).

import { cp, mkdir, rm } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(__dirname, '../..')
const publicDir = resolve(repoRoot, 'public')

// Each entry: [sourcePathRelativeToRepoRoot, destPathRelativeToPublic]
const copies = [
  ['specification/schema', 'specification/schema'],
]

for (const [src, dest] of copies) {
  const srcAbs = resolve(repoRoot, src)
  const destAbs = resolve(publicDir, dest)
  await rm(destAbs, { recursive: true, force: true })
  await mkdir(dirname(destAbs), { recursive: true })
  // Skip markdown files — VitePress renders the in-place markdown sources;
  // we only want the non-md reference artifacts (JSON schemas) copied as
  // static assets.
  await cp(srcAbs, destAbs, {
    recursive: true,
    filter: (p) => !p.includes('/node_modules/') && !p.endsWith('.md'),
  })
  console.log(`copy-static: ${src} → public/${dest}`)
}
