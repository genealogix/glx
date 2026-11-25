// SPDX-FileCopyrightText: 2025 Oracynth, Inc.
// SPDX-License-Identifier: Apache-2.0

import { createHash } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const modulePath = 'github.com/modelcontextprotocol/go-sdk'
const version = 'v1.8.0'
const expectedSums = [
  `${modulePath} ${version} h1:KIvahhYqwtbeniWVPs3TcXEA7b8jEtwfBpOTAI+Urx4=`,
  `${modulePath} ${version}/go.mod h1:dL7u98E/zjJTGzEq+j30jQ8K2k1mb6LeAH4inEcSGts=`,
].sort()
const licenseSHA256 = 'af679003d933f045393a6a029f43da113f9ae364eac651d9ae268392985580f5'

// The pinned dependency-review action ignores versions when matching license
// exception PURLs. Fail closed if this reviewed package, content, or notice
// changes; a version suffix in the workflow alone would not limit the exception.
export function verifyMCPLicenseException({ goMod, goSum, notice, workflow, otherManifests = [] }) {
  for (const { name, contents } of otherManifests) {
    if (/(^|\/)go\.work(?:\.sum)?$/.test(name) || contents.toLowerCase().includes(modulePath)) {
      throw new Error('Review the MCP SDK license exception before adding workspaces or other SDK manifests')
    }
  }
  const actionPins = workflow.split(/\r?\n/)
    .map(line => line.split('#')[0].trim())
    .filter(line => line.startsWith('uses: actions/dependency-review-action@'))
  if (actionPins.length !== 1 || actionPins[0] !== 'uses: actions/dependency-review-action@bbc44d14a204704ccc493733616cd1e93a0321ec') {
    throw new Error('Review the MCP SDK license exception when updating the dependency-review action')
  }
  // Git checkouts may use CRLF; preserve license content, not newline encoding.
  notice = notice.replace(/\r\n/g, '\n')
  const mentions = goMod.split(/\r?\n/)
    .map(line => line.split('//')[0].trim().replace(/\s+/g, ' '))
    .filter(line => line.toLowerCase().includes(modulePath))
  const expectedRequirement = `${modulePath} ${version}`
  if (mentions.length !== 1 || ![expectedRequirement, `require ${expectedRequirement}`].includes(mentions[0])) {
    throw new Error('Review the MCP SDK license exception before changing or replacing the pinned SDK requirement')
  }
  const sums = goSum.split(/\r?\n/).filter(line => line.toLowerCase().startsWith(`${modulePath} `)).sort()
  if (JSON.stringify(sums) !== JSON.stringify(expectedSums)) {
    throw new Error('Review the MCP SDK license exception: SDK checksums differ from the reviewed module')
  }
  const begin = `BEGIN UPSTREAM LICENSE: ${modulePath} ${version}\n`
  const end = `END UPSTREAM LICENSE: ${modulePath} ${version}`
  const start = notice.indexOf(begin)
  const finish = notice.indexOf(end, start + begin.length)
  if (start < 0 || finish < 0 || notice.indexOf(end) !== finish || notice.indexOf(begin, start + begin.length) !== -1 || notice.indexOf(end, finish + end.length) !== -1) {
    throw new Error('Preserve the reviewed MCP SDK license in the release-shipped NOTICE file')
  }
  const license = notice.slice(start + begin.length, finish)
  if (createHash('sha256').update(license).digest('hex') !== licenseSHA256) {
    throw new Error('Review the MCP SDK license exception: upstream license text in NOTICE changed')
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const root = dirname(fileURLToPath(new URL('../go.mod', import.meta.url)))
    const tracked = execFileSync('git', ['ls-files', '-z'], { cwd: root, encoding: 'utf8' }).split('\0')
    const otherManifests = tracked
      .filter(name => name !== 'go.mod' && name !== 'go.sum' && (name.endsWith('.mod') || name.endsWith('.sum') || /(^|\/)go\.work$/.test(name)))
      .map(name => ({ name, contents: readFileSync(resolve(root, name), 'utf8') }))
    verifyMCPLicenseException({
      otherManifests,
      workflow: readFileSync(new URL('../.github/workflows/dependency-review.yml', import.meta.url), 'utf8'),
      goMod: readFileSync(new URL('../go.mod', import.meta.url), 'utf8'),
      goSum: readFileSync(new URL('../go.sum', import.meta.url), 'utf8'),
      notice: readFileSync(new URL('../NOTICE', import.meta.url), 'utf8'),
    })
    console.log('MCP SDK license exception matches the reviewed v1.8.0 module and upstream notice')
  } catch (error) {
    console.error(error.message)
    process.exitCode = 1
  }
}
