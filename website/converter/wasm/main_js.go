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

	input, destination, prepare := args[0], args[1].String(), args[2]
	data := make([]byte, input.Length())
	js.CopyBytesToGo(data, input)

	output, report, err := convert(data, destination, func() { prepare.Invoke() })
	if err != nil {
		return map[string]any{errorKey: err.Error()}
	}
	if destination != singleFileGLX && destination != multiFileGLX {
		prepare.Invoke()
	}

	metadata, err := json.Marshal(report)
	if err != nil {
		return map[string]any{errorKey: "Could not prepare the conversion summary."}
	}
	result := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(result, output)

	return map[string]any{"output": result, "summary": string(metadata)}
}
