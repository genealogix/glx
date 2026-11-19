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
	"context"
)

// runContextPhases cooperatively stops between existing synchronous operations.
// A phase is never abandoned in a background goroutine; it finishes before the
// cancellation error is returned and the backend's serialization gate is freed.
func runContextPhases(ctx context.Context, phases ...func()) error {
	for _, phase := range phases {
		if err := ctx.Err(); err != nil {
			return err
		}
		phase()
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	return ctx.Err()
}
