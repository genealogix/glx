import { execFileSync } from 'node:child_process'
import { copyFile, mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('../../', import.meta.url))
const generated = join(root, 'website/converter/generated')
const goVersion = (await readFile(join(root, 'go.mod'), 'utf8')).match(/^go (\S+)$/m)[1]
const env = { ...process.env, GOTOOLCHAIN: `go${goVersion}`, GOOS: 'js', GOARCH: 'wasm', CGO_ENABLED: '0' }
const go = (args) => execFileSync('go', args, { cwd: root, env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] }).trim()
const bridge = './website/converter/wasm'

await mkdir(generated, { recursive: true })
go(['build', '-mod=readonly', '-trimpath', '-buildvcs=false', '-ldflags=-s -w', '-o', join(generated, 'converter.wasm'), bridge])
const goRoot = go(['env', 'GOROOT'])
const runtimePath = join(generated, 'wasm_exec.js')
// Go toolchains installed as modules contain read-only files.
await rm(runtimePath, { force: true })
await copyFile(join(goRoot, 'lib/wasm/wasm_exec.js'), runtimePath)

// Publish the license notices with the binary and the matching Go runtime.
const modules = go(['list', '-mod=readonly', '-deps', '-f', '{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Dir}}{{end}}{{end}}', bridge])
  .split('\n').filter(Boolean).sort()
const sources = new Map([
  ['Genealogix', root],
  [`Go ${goVersion}`, goRoot],
  ...modules.map((line) => line.split('\t')),
])
const notices = []
for (const [name, directory] of sources) {
  const files = (await readdir(directory)).filter((file) => /^(LICENSE|NOTICE|PATENTS)(\.txt)?$/.test(file)).sort()
  for (const file of files) {
    notices.push(`=== ${name} / ${file} ===\n\n${await readFile(join(directory, file), 'utf8')}`)
  }
}
await writeFile(join(root, 'public/converter-licenses.txt'), notices.join('\n\n'))
console.log(`Built the GEDCOM and GLX converter with Go ${goVersion}`)
