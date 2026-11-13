<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'

const stages = {
  loading: 'Loading the converter…',
  reading: 'Reading the selected file…',
  converting: 'Converting genealogy records…',
  preparing: 'Preparing the download…',
  done: 'Ready to download.',
  cancelled: 'Conversion cancelled.',
}
const diagnosticBatch = 25
const ready = ref(false)
const direction = ref('import')
const version = ref('5.5.1')
const fileInput = ref(null)
const file = ref(null)
const stage = ref('')
const error = ref('')
const download = ref(null)
const diagnosticsOpen = ref(false)
const diagnosticLimit = ref(diagnosticBatch)
let worker = null

const isImport = computed(() => direction.value === 'import')
const busy = computed(() => ['loading', 'reading', 'converting', 'preparing'].includes(stage.value))
const diagnostics = computed(() => download.value?.summary.diagnostics ?? [])
const errors = computed(() => diagnostics.value.filter((entry) => entry.isError).length)
const warnings = computed(() => diagnostics.value.length - errors.value)
const status = computed(() => {
  if (error.value) return error.value
  if (stage.value === 'done' && errors.value) return 'Ready to download. Some records could not be converted.'
  return stages[stage.value] || (file.value ? 'File selected.' : '')
})
const number = new Intl.NumberFormat('en')

function stopWorker() {
  if (worker) {
    worker.onmessage = worker.onerror = worker.onmessageerror = null
    worker.terminate()
    worker = null
  }
}

function clearResult() {
  if (download.value) URL.revokeObjectURL(download.value.url)
  download.value = null
  diagnosticsOpen.value = false
  diagnosticLimit.value = diagnosticBatch
  stage.value = ''
  error.value = ''
}

function clearSelection() {
  clearResult()
  file.value = null
  if (fileInput.value) fileInput.value.value = ''
}

function selectFile(event) {
  clearResult()
  file.value = event.target.files[0] ?? null
}

function cancel() {
  stopWorker()
  stage.value = 'cancelled'
}

function reset() {
  stopWorker()
  clearSelection()
  direction.value = 'import'
  version.value = '5.5.1'
}

function fail(message) {
  stopWorker()
  stage.value = ''
  error.value = message
}

function showDownload(message) {
  const extension = isImport.value ? '.glx' : '.ged'
  const baseName = file.value.name.replace(/\.(ged(com)?|glx)$/i, '') || 'archive'
  const blob = new Blob([message.output], { type: 'application/octet-stream' })
  download.value = {
    url: URL.createObjectURL(blob),
    name: baseName + extension,
    summary: message.summary,
  }
  stage.value = 'done'
}

function start() {
  if (!file.value || busy.value) return
  clearResult()
  if (file.value.size === 0) {
    fail('The selected file is empty.')
    return
  }
  if (/\.(gdz|gedzip|zip)$/i.test(file.value.name)) {
    fail('ZIP and GEDZIP files require the GLX CLI.')
    return
  }

  const workerFailure = 'The converter stopped. Try again or use the GLX CLI.'
  stage.value = 'loading'
  try {
    const run = new Worker(new URL('./converter-worker.js', import.meta.url), {
      type: 'module',
      name: 'glx-converter',
    })
    worker = run
    run.onerror = (event) => {
      event.preventDefault()
      if (worker === run) fail(workerFailure)
    }
    run.onmessageerror = () => {
      if (worker === run) fail(workerFailure)
    }
    run.onmessage = ({ data }) => {
      if (worker !== run) return
      if (data.type === 'stage') {
        stage.value = data.stage
        return
      }
      if (data.type === 'result') {
        stopWorker()
        try {
          showDownload(data)
        } catch {
          fail('Could not prepare the download. Try again or use the GLX CLI.')
        }
        return
      }
      fail(data.type === 'error' ? data.message : workerFailure)
    }
    run.postMessage({ file: file.value, destination: isImport.value ? 'glx' : version.value })
  } catch {
    fail(workerFailure)
  }
}

onMounted(() => {
  ready.value = true
  window.addEventListener('pagehide', reset)
})
onUnmounted(() => {
  window.removeEventListener('pagehide', reset)
  reset()
})
</script>

<template>
  <section
    class="converter"
    aria-label="File conversion"
  >
    <form @submit.prevent="start">
      <fieldset
        class="conversion-fields"
        :disabled="!ready || busy"
      >
        <div class="conversion-options">
          <label>
            Conversion direction
            <select
              v-model="direction"
              @change="clearSelection"
            >
              <option value="import">GEDCOM → GLX</option>
              <option value="export">GLX → GEDCOM</option>
            </select>
          </label>
          <label v-if="!isImport">
            GEDCOM version
            <select
              v-model="version"
              @change="clearResult"
            >
              <option value="5.5.1">5.5.1</option>
              <option value="7.0">7.0</option>
            </select>
          </label>
        </div>
        <label>
          Source file
          <input
            ref="fileInput"
            type="file"
            :accept="isImport ? '.ged,.gedcom' : '.glx'"
            @change="selectFile"
          >
        </label>
      </fieldset>
      <div class="actions">
        <button
          class="button primary"
          type="submit"
          :disabled="!file || busy"
        >
          Convert file
        </button>
        <button
          v-if="busy"
          class="button"
          type="button"
          @click="cancel"
        >
          Cancel
        </button>
        <button
          class="button"
          type="button"
          :disabled="!ready"
          @click="reset"
        >
          Reset
        </button>
      </div>
    </form>

    <div class="status">
      <p
        :class="{ error }"
        role="status"
      >
        {{ status }}
      </p>
      <progress
        v-if="busy"
        aria-label="Conversion in progress"
      />
    </div>

    <div
      v-if="download"
      class="result"
    >
      <div class="result-heading">
        <div>
          <p class="result-title">
            Conversion result
          </p>
          <p class="result-format">
            GEDCOM {{ download.summary.version }} {{ isImport ? '→ GLX' : 'output' }}
          </p>
        </div>
        <a
          class="button primary"
          :href="download.url"
          :download="download.name"
        >Download {{ isImport ? 'GLX' : 'GEDCOM' }}</a>
      </div>
      <dl
        class="counts"
        aria-label="Records converted"
      >
        <div
          v-for="item in download.summary.counts"
          :key="item.label"
        >
          <dt>{{ item.label }}</dt>
          <dd>{{ number.format(item.value) }}</dd>
        </div>
      </dl>
      <details
        v-if="diagnostics.length"
        @toggle="diagnosticsOpen = $event.target.open"
      >
        <summary>{{ errors ? `${number.format(errors)} record errors · ` : '' }}{{ number.format(warnings) }} warnings</summary>
        <template v-if="diagnosticsOpen">
          <ol>
            <li
              v-for="(entry, index) in diagnostics.slice(0, diagnosticLimit)"
              :key="index"
            >
              <strong v-if="entry.isError">Record error</strong> {{ entry.message }}
            </li>
          </ol>
          <button
            v-if="diagnostics.length > diagnosticLimit"
            class="button"
            type="button"
            @click="diagnosticLimit += diagnosticBatch"
          >
            Show more
          </button>
        </template>
      </details>
    </div>
  </section>
</template>

<style scoped>
.converter {
  margin: 28px 0 36px;
  padding: clamp(16px, 4vw, 28px);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
}

.conversion-fields {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.conversion-options {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 20px;
}

.conversion-options label {
  flex: 1 1 155px;
}

label {
  display: grid;
  gap: 5px;
  font-size: 14px;
  font-weight: 600;
}

select,
input[type='file'] {
  width: 100%;
  min-height: 42px;
  padding: 6px 10px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 7px;
  background-color: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  font: inherit;
  font-weight: 400;
}

select {
  appearance: auto;
}

input::file-selector-button {
  margin-inline-end: 10px;
  padding: 3px 10px;
  border: 0;
  border-radius: 5px;
  background: var(--vp-c-default-soft);
  color: var(--vp-c-text-1);
  font: inherit;
  cursor: pointer;
}

.actions,
.result-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.actions {
  margin-top: 20px;
}

.button {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
  padding: 5px 14px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 7px;
  background: var(--vp-c-bg);
  color: var(--vp-c-text-1);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
}

.button.primary {
  border-color: transparent;
  background: var(--vp-button-brand-bg);
  color: var(--vp-button-brand-text);
}

.button:not(:disabled):hover {
  border-color: var(--vp-c-brand-2);
}

.button.primary:not(:disabled):hover {
  background: var(--vp-button-brand-hover-bg);
  color: var(--vp-button-brand-hover-text);
}

.button:disabled {
  opacity: 0.55;
  cursor: default;
}

.converter :focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 3px;
}

.status p:not(:empty) {
  margin: 16px 0 0;
  color: var(--vp-c-text-2);
  overflow-wrap: anywhere;
}

.status p.error {
  color: var(--vp-c-danger-1);
}

progress {
  width: 100%;
  height: 8px;
  accent-color: var(--vp-c-brand-1);
}

.result {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--vp-c-divider);
}

.result-heading {
  justify-content: space-between;
}

.result-title,
.result-format {
  margin: 0;
}

.result-title {
  font-size: 17px;
  font-weight: 600;
}

.result-format,
dt {
  color: var(--vp-c-text-2);
  font-size: 13px;
}

.counts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
  gap: 12px;
  margin: 16px 0 0;
}

.counts div {
  display: flex;
  flex-direction: column-reverse;
  padding: 8px 12px;
  border-radius: 7px;
  background: var(--vp-c-bg);
}

dd {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

details {
  margin-top: 16px;
  overflow-wrap: anywhere;
}

summary {
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
</style>
