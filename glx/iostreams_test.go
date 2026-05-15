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
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemIOStreams_Default(t *testing.T) {
	t.Cleanup(func() { quietOutput = false })
	quietOutput = false

	streams := SystemIOStreams()
	assert.Same(t, os.Stdout, streams.Out, "Out should be os.Stdout when quiet flag is unset")
	assert.Same(t, os.Stderr, streams.ErrOut, "ErrOut should always be os.Stderr")
}

func TestSystemIOStreams_Quiet(t *testing.T) {
	t.Cleanup(func() { quietOutput = false })
	quietOutput = true

	streams := SystemIOStreams()
	assert.Equal(t, io.Discard, streams.Out, "Out should be io.Discard when quiet flag is set")
	assert.Same(t, os.Stderr, streams.ErrOut, "ErrOut should remain os.Stderr even when quiet")
}
