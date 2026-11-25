// SPDX-FileCopyrightText: 2025 Oracynth, Inc.
// SPDX-License-Identifier: Apache-2.0

import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { verifyMCPLicenseException } from './check-mcp-license-exception.mjs'

const files = {
  workflow: readFileSync(new URL('../.github/workflows/dependency-review.yml', import.meta.url), 'utf8'),
  goMod: readFileSync(new URL('../go.mod', import.meta.url), 'utf8'),
  goSum: readFileSync(new URL('../go.sum', import.meta.url), 'utf8'),
  notice: readFileSync(new URL('../NOTICE', import.meta.url), 'utf8'),
}
const modulePath = 'github.com/modelcontextprotocol/go-sdk'

test('reviewed MCP module and upstream notice pass the guard', () => {
  assert.doesNotThrow(() => verifyMCPLicenseException(files))
  const run = spawnSync(process.execPath, [fileURLToPath(new URL('./check-mcp-license-exception.mjs', import.meta.url))], { encoding: 'utf8' })
  assert.equal(run.status, 0, run.stderr)
})

for (const [name, changed] of [
  ['SDK upgrade', { goMod: files.goMod.replace(`${modulePath} v1.8.0`, `${modulePath} v1.9.0`) }],
  ['removed requirement', { goMod: files.goMod.replace(`${modulePath} v1.8.0`, '') }],
  ['replacement', { goMod: `${files.goMod}\nreplace ${modulePath} => ./unreviewed-sdk\n` }],
  ['changed checksum', { goSum: files.goSum.replace('h1:KIvahhYqwtbeniWVPs3TcXEA7b8jEtwfBpOTAI+Urx4=', 'h1:unreviewed') }],
  ['another SDK version', { goSum: `${files.goSum}${modulePath} v1.9.0 h1:unreviewed\n` }],
  ['second module requirement', { otherManifests: [{ name: 'ci-tools/go.mod', contents: `require ${modulePath} v1.9.0` }] }],
  ['second module checksum', { otherManifests: [{ name: 'future/go.sum', contents: `${modulePath} v1.9.0 h1:unreviewed` }] }],
  ['workspace', { otherManifests: [{ name: 'go.work', contents: 'go 1.27.4\nuse ../unreviewed-sdk' }] }],
  ['workspace sums', { otherManifests: [{ name: 'go.work.sum', contents: '' }] }],
  ['changed action semantics', { workflow: files.workflow.replace('bbc44d14a204704ccc493733616cd1e93a0321ec', '0000000000000000000000000000000000000000') }],
  ['case-variant SDK requirement', { goMod: `${files.goMod}\nrequire github.com/ModelContextProtocol/go-sdk v1.9.0\n` }],
  ['case-variant secondary module', { otherManifests: [{ name: 'other/go.mod', contents: 'require github.com/ModelContextProtocol/go-sdk v1.9.0' }] }],
  ['duplicate notice', { notice: files.notice + files.notice }],
  ['orphaned notice end', { notice: `END UPSTREAM LICENSE: ${modulePath} v1.8.0\n${files.notice}` }],
  ['removed notice', { notice: '' }],
  ['changed license text', { notice: files.notice.replace('Documentation contributions (excluding specifications)', 'Documentation contributions') }],
]) {
  test(`${name} requires a new license decision`, () => {
    assert.notDeepEqual({ ...files, ...changed }, files, 'fixture mutation must not be vacuous')
    assert.throws(() => verifyMCPLicenseException({ ...files, ...changed }))
  })
}

test('workflow runs the guard before its package-scoped exception', () => {
  const workflow = readFileSync(new URL('../.github/workflows/dependency-review.yml', import.meta.url), 'utf8')
  const guard = workflow.indexOf('run: node scripts/check-mcp-license-exception.mjs')
  const action = workflow.indexOf('uses: actions/dependency-review-action@')
  assert.ok(guard >= 0 && action > guard)
  assert.match(workflow, /allow-dependencies-licenses: pkg:golang\/github\.com\/modelcontextprotocol\/go-sdk@1\.8\.0/)
  assert.match(workflow, /vulnerability-check: true/)
  assert.match(workflow, /license-check: true/)
})

test('line endings and unrelated Go manifests do not broaden the exception', () => {
  assert.doesNotThrow(() => verifyMCPLicenseException({
    ...files,
    goMod: files.goMod.replace(/\n/g, '\r\n'),
    goSum: files.goSum.replace(/\n/g, '\r\n'),
    notice: files.notice.replace(/\n/g, '\r\n'),
    otherManifests: [{ name: 'ci-tools/go.mod', contents: 'module example.invalid/tools\n' }],
  }))
})
