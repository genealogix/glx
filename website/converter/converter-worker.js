import './generated/wasm_exec.js'
import wasmUrl from './generated/converter.wasm?url&no-inline'

// Each worker accepts one job. The page terminates it after a result, on
// failure, or on Cancel so Go's heap is released between conversions.
self.addEventListener('message', async ({ data: { file, destination } }) => {
  let stage = 'loading'
  const setStage = (value) => {
    stage = value
    self.postMessage({ type: 'stage', stage })
  }

  try {
    const go = new self.Go()
    const response = await fetch(wasmUrl, { credentials: 'same-origin' })
    if (!response.ok) throw new Error('WASM request failed')
    const { instance } = await WebAssembly.instantiateStreaming(response, go.importObject)
    const runtime = go.run(instance)
    // Go registers the bridge synchronously before waiting for calls.
    const reportExit = () => {
      self.postMessage({ type: 'error', message: 'The converter stopped. Try again or use the GLX CLI.' })
    }
    void runtime.then(reportExit, reportExit)

    setStage('reading')
    const input = new Uint8Array(await file.arrayBuffer())
    setStage('converting')
    const result = self.convertGLX(input, destination, () => setStage('preparing'))
    if (result.error) {
      self.postMessage({ type: 'error', message: result.error })
      return
    }

    const output = result.output.buffer
    self.postMessage({ type: 'result', output, summary: JSON.parse(result.summary) }, [output])
  } catch {
    const messages = {
      loading: 'Could not load the converter. Check your connection and try again.',
      reading: 'Could not read the selected file. Choose it again and retry.',
      converting: 'The converter stopped. Try again or use the GLX CLI.',
      preparing: 'Could not prepare the download. Try again or use the GLX CLI.',
    }
    self.postMessage({ type: 'error', message: messages[stage] })
  }
}, { once: true })
