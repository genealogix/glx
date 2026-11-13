import { expect, test } from '@playwright/test'
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { inflateRawSync } from 'node:zlib'

const fromRepo = (path) => fileURLToPath(new URL(`../../${path}`, import.meta.url))
const shakespeare = fromRepo('glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged')
const sameSexMarriage = fromRepo('glx/testdata/gedcom/7.0/same-sex-marriage/same-sex-marriage.ged')
const singleArchive = fromRepo('docs/examples/single-file/archive.glx')
const converterPath = '/convert.html'

async function selectAndConvert(page, file) {
  await page.getByLabel('Source file').setInputFiles(file)
  await page.getByRole('button', { name: 'Convert file' }).click()
}

async function download(page, format, name) {
  const link = page.getByRole('link', { name: `Download ${format}` })
  const [result] = await Promise.all([page.waitForEvent('download'), link.click()])
  expect(result.suggestedFilename()).toBe(name)
  return readFile(await result.path())
}

function expectCount(page, label, value) {
  return expect(page.locator('.counts div').filter({ has: page.getByText(label, { exact: true }) }).locator('dd')).toHaveText(value)
}

for (const [fixture, version, baseName, people, person] of [
  [shakespeare, '5.5.1', 'shakespeare', '31', 'William Shakespeare'],
  [sameSexMarriage, '7.0', 'same-sex-marriage', '2', 'Richard Roe'],
]) {
  test(`imports GEDCOM ${version} and reimports the downloaded GLX`, async ({ page }) => {
    await page.goto(converterPath)
    await selectAndConvert(page, fixture)
    await expectCount(page, 'People', people)
    const glx = await download(page, 'GLX', `${baseName}.glx`)
    expect(glx.toString()).toContain(person)

    await page.getByLabel('Conversion direction').selectOption('export')
    await page.getByLabel('GEDCOM version').selectOption(version)
    await selectAndConvert(page, { name: `${baseName}.glx`, mimeType: 'application/octet-stream', buffer: glx })
    await expectCount(page, 'People', people)
    const gedcom = await download(page, 'GEDCOM', `${baseName}.ged`)
    expect(gedcom.toString()).toContain(`2 VERS ${version}`)
    expect(gedcom.toString()).toContain('0 TRLR')

    await page.getByLabel('Conversion direction').selectOption('import')
    await selectAndConvert(page, { name: `${baseName}.ged`, mimeType: 'application/octet-stream', buffer: gedcom })
    await expectCount(page, 'People', people)
    const reimported = await download(page, 'GLX', `${baseName}.glx`)
    expect(reimported.toString()).toContain(person)
  })
}

test('exports an existing single-file archive and reports a library warning', async ({ page }) => {
  await page.goto(converterPath)
  await page.getByLabel('Conversion direction').selectOption('export')
  await selectAndConvert(page, singleArchive)
  const original = await download(page, 'GEDCOM', 'archive.ged')
  expect(original.toString()).toContain('2 VERS 5.5.1')

  await page.getByLabel('GEDCOM version').selectOption('7.0')
  await expect(page.getByRole('link', { name: 'Download GEDCOM' })).toHaveCount(0)
  await selectAndConvert(page, {
    name: 'unattached-event.glx',
    mimeType: 'application/octet-stream',
    buffer: Buffer.from('events:\n  event-unattached:\n    type: birth\n'),
  })
  await page.locator('summary').filter({ hasText: 'warnings' }).click()
  await expect(page.locator('.converter li').first()).toContainText('event-unattached')
  const gedcom = await download(page, 'GEDCOM', 'unattached-event.ged')
  expect(gedcom.toString()).toContain('2 VERS 7.0')
})

test('switches GEDCOM import between a GLX file and a multi-file ZIP', async ({ page }) => {
  await page.goto(converterPath)
  const format = page.getByLabel('GLX output')
  await expect(format).toHaveValue('glx')
  await selectAndConvert(page, shakespeare)
  await expect(page.getByRole('link', { name: 'Download GLX' })).toBeVisible()

  await format.selectOption('zip')
  await expect(page.getByRole('link', { name: 'Download GLX' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Convert file' }).click()
  await expectCount(page, 'People', '31')
  const zip = await download(page, 'ZIP', 'shakespeare.zip')

  // Inspect the small fixture's ZIP central directory and inflate one file.
  const end = zip.length - 22
  expect(zip.readUInt32LE(end)).toBe(0x06054b50)
  const entries = zip.readUInt16LE(end + 10)
  const names = []
  let offset = zip.readUInt32LE(end + 16)
  let person
  for (let index = 0; index < entries; index++) {
    expect(zip.readUInt32LE(offset)).toBe(0x02014b50)
    const nameLength = zip.readUInt16LE(offset + 28)
    const name = zip.toString('utf8', offset + 46, offset + 46 + nameLength)
    names.push(name)
    if (name === 'persons/person-1.glx') {
      const size = zip.readUInt32LE(offset + 20)
      const header = zip.readUInt32LE(offset + 42)
      const dataStart = header + 30 + zip.readUInt16LE(header + 26) + zip.readUInt16LE(header + 28)
      person = inflateRawSync(zip.subarray(dataStart, dataStart + size)).toString()
    }
    offset += 46 + nameLength + zip.readUInt16LE(offset + 30) + zip.readUInt16LE(offset + 32)
  }
  expect(names).toContain('metadata.glx')
  expect(names).toContain('vocabularies/event-types.glx')
  expect(names.filter((name) => name.startsWith('persons/'))).toHaveLength(31)
  expect(person).toContain('Shakespeare')

  await page.getByRole('button', { name: 'Reset' }).click()
  await expect(page.getByRole('link', { name: 'Download ZIP' })).toHaveCount(0)
  await expect(format).toHaveValue('glx')
})

test('recovers from malformed files and rejects GEDZIP', async ({ page }) => {
  await page.goto(converterPath)
  await selectAndConvert(page, { name: 'wrong.ged', mimeType: 'text/plain', buffer: Buffer.from('0 unrelated') })
  await expect(page.getByRole('status')).toContainText('0 HEAD')
  await expect(page.getByRole('link', { name: 'Download GLX' })).toHaveCount(0)
  await selectAndConvert(page, { name: 'utf16.ged', mimeType: 'text/plain', buffer: Buffer.from('0 HEAD\n0 TRLR\n', 'utf16le') })
  await expect(page.getByRole('status')).toContainText('Save or export the file as UTF-8')
  await selectAndConvert(page, { name: 'packed.gdz', mimeType: 'application/zip', buffer: Buffer.from('archive') })
  await expect(page.getByRole('status')).toContainText('GEDZIP files require the GLX CLI')
  await selectAndConvert(page, shakespeare)
  await expectCount(page, 'People', '31')

  await page.getByLabel('Conversion direction').selectOption('export')
  await selectAndConvert(page, { name: 'other.glx', mimeType: 'text/plain', buffer: Buffer.from('unrelated: true') })
  await expect(page.getByRole('status')).toContainText('expected a single-file GLX archive')
  await selectAndConvert(page, { name: 'broken.glx', mimeType: 'text/plain', buffer: Buffer.from('persons: [') })
  await expect(page.getByRole('status')).toContainText('could not read GLX')
  await selectAndConvert(page, singleArchive)
  await expect(page.getByRole('link', { name: 'Download GEDCOM' })).toBeVisible()
})

test('cancels, retries with a fresh worker and releases the result on reset', async ({ page }) => {
  await page.addInitScript(() => {
    const WorkerConstructor = window.Worker
    window.workerRuns = []
    window.Worker = class extends WorkerConstructor {
      constructor(...args) {
        super(...args)
        this.run = { stopped: false }
        window.workerRuns.push(this.run)
      }
      terminate() {
        this.run.stopped = true
        return super.terminate()
      }
    }
    const revokeObjectURL = URL.revokeObjectURL
    window.revokedUrls = []
    URL.revokeObjectURL = (url) => {
      window.revokedUrls.push(url)
      revokeObjectURL(url)
    }
  })
  await page.goto(converterPath)
  await page.getByLabel('Source file').setInputFiles(shakespeare)
  await page.evaluate(() => {
    document.querySelector('button[type=submit]').click()
    // Wait for Vue to render the Cancel button while the worker starts up.
    queueMicrotask(() => [...document.querySelectorAll('.converter button')].find((button) => button.textContent.trim() === 'Cancel').click())
  })
  await expect(page.getByRole('status')).toHaveText('Conversion cancelled.')
  await expect(page.getByRole('link', { name: 'Download GLX' })).toHaveCount(0)

  await page.getByRole('button', { name: 'Convert file' }).click()
  const link = page.getByRole('link', { name: 'Download GLX' })
  await expect(link).toBeVisible()
  const url = await link.getAttribute('href')
  await page.getByRole('button', { name: 'Reset' }).click()
  await expect(link).toHaveCount(0)
  await expect(page.getByLabel('Source file')).toHaveValue('')
  expect(await page.evaluate(() => window.workerRuns)).toEqual([{ stopped: true }, { stopped: true }])
  expect(await page.evaluate(() => window.revokedUrls)).toContain(url)
})

test('loads the converter on demand in a document without analytics', async ({ page }) => {
  const requests = []
  page.on('request', (request) => requests.push({ url: request.url(), method: request.method() }))
  await page.route('https://www.googletagmanager.com/**', (route) => route.fulfill({
    contentType: 'application/javascript',
    body: 'window.analyticsStarted = true',
  }))
  await page.goto('/quickstart.html')
  await expect.poll(() => page.evaluate(() => window.analyticsStarted)).toBe(true)
  await page.getByRole('link', { name: 'browser converter' }).click()
  await expect(page.getByLabel('Source file')).toBeEnabled()
  expect(await page.evaluate(() => window.analyticsStarted)).toBeUndefined()
  await expect(page.locator('script[src*="googletagmanager"]')).toHaveCount(0)
  const beforeSelection = requests.length
  await page.getByLabel('Source file').setInputFiles({
    name: 'private-filename.ged',
    mimeType: 'text/plain',
    buffer: Buffer.from('0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 NAME Private /Person/\n1 OBJE @M1@\n0 @M1@ OBJE\n1 FILE https://example.invalid/private-portrait.jpg\n2 FORM image/jpeg\n0 _PRIVATE\n0 TRLR\n'),
  })
  expect(requests.some(({ url }) => /converter-worker-.*\.js|\.wasm$/.test(url))).toBe(false)
  await page.getByRole('button', { name: 'Convert file' }).click()
  const glx = await download(page, 'GLX', 'private-filename.glx')
  expect(glx.toString()).toContain('https://example.invalid/private-portrait.jpg')
  await page.locator('summary').filter({ hasText: 'warnings' }).click()
  await expect(page.locator('.converter li')).toContainText(['Top-level extension tag not preserved'])
  for (const { url, method } of requests.slice(beforeSelection)) {
    expect(method).toBe('GET')
    const resource = new URL(url)
    expect(resource.origin).toBe('http://127.0.0.1:4173')
    expect(resource.search).toBe('')
    if (resource.protocol !== 'blob:') {
      // VitePress can prefetch documentation JS alongside the converter.
      expect(resource.pathname).toMatch(/^\/assets\/[\w.-]+\.(?:js|wasm)$/)
    }
  }
})

test('offers a retry after a WASM load failure or worker script failure', async ({ page, browserName }) => {
  test.skip(browserName !== 'chromium', 'Worker network fault injection uses Chromium request routing.')
  await page.goto(converterPath)
  await page.route('**/converter-*.wasm', (route) => route.abort())
  await selectAndConvert(page, shakespeare)
  await expect(page.getByRole('status')).toContainText('Could not load the converter')
  await page.unroute('**/converter-*.wasm')

  await page.route('**/converter-worker-*.js', (route) => route.fulfill({
    contentType: 'application/javascript',
    body: 'throw new Error("simulated worker failure")',
  }))
  await page.getByRole('button', { name: 'Convert file' }).click()
  await expect(page.getByRole('status')).toContainText('The converter stopped')
  await page.unroute('**/converter-worker-*.js')
  await page.getByRole('button', { name: 'Convert file' }).click()
  await expectCount(page, 'People', '31')
})
