// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"syscall/js"
)

const errorKey = "error"

func main() {
	// The worker owns this callback and terminates the runtime after one run.
	js.Global().Set("convertGLX", js.FuncOf(convertFromWorker))
	select {}
}

func convertFromWorker(_ js.Value, args []js.Value) (response any) {
	defer func() {
		if recover() != nil {
			response = map[string]any{errorKey: "The converter stopped while processing these records. Check the file or try the GLX CLI."}
		}
	}()

	// The worker hands the input over through a getter that drops its own
	// reference, so JavaScript can reclaim the file once Go has a copy.
	takeInput, destination, prepare := args[0], args[1].String(), args[2]
	data := copyInput(takeInput.Invoke())

	output := &jsChunks{chunks: js.Global().Get("Array").New()}
	report, err := convert(data, destination, func() { prepare.Invoke() }, output)
	if err != nil {
		return map[string]any{errorKey: err.Error()}
	}
	output.flush()
	if destination != singleFileGLX && destination != multiFileGLX {
		prepare.Invoke()
	}

	metadata, err := json.Marshal(report)
	if err != nil {
		return map[string]any{errorKey: "Could not prepare the conversion summary."}
	}

	return map[string]any{"output": output.chunks, "summary": string(metadata)}
}

// copyInput copies a Uint8Array into Go. Go keeps no reference to the array
// once this returns, so its memory is the browser's to reclaim.
func copyInput(input js.Value) []byte {
	data := make([]byte, input.Length())
	js.CopyBytesToGo(data, input)

	return data
}

// jsChunkSize bounds the output held in Go memory at once.
const jsChunkSize = 4 << 20

// jsChunks copies the output into a JavaScript array of Uint8Array chunks as
// it is written, so the whole result never sits in the Go heap beside its
// JavaScript copy. The page joins the chunks into one Blob.
type jsChunks struct {
	chunks js.Value
	buffer []byte
}

func (w *jsChunks) Write(data []byte) (int, error) {
	if len(w.buffer)+len(data) > jsChunkSize {
		w.flush()
	}
	if len(data) >= jsChunkSize {
		w.push(data)
	} else {
		if w.buffer == nil {
			w.buffer = make([]byte, 0, jsChunkSize)
		}
		w.buffer = append(w.buffer, data...)
	}

	return len(data), nil
}

func (w *jsChunks) flush() {
	if len(w.buffer) > 0 {
		w.push(w.buffer)
		w.buffer = w.buffer[:0]
	}
}

func (w *jsChunks) push(data []byte) {
	chunk := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(chunk, data)
	w.chunks.Call("push", chunk)
}
