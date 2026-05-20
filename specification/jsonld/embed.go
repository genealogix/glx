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

// Package jsonld provides the embedded canonical JSON-LD @context for GENEALOGIX.
package jsonld

import (
	_ "embed"
)

// ContextBytes contains the embedded glx-context.jsonld file. It is the
// canonical mapping of GLX entity types and properties to Schema.org classes
// (with a glx: namespace for GLX-native concepts that Schema.org doesn't
// cover) and is embedded inline in every `glx export --format jsonld` output
// so emitted documents are self-contained.
//
//go:embed glx-context.jsonld
var ContextBytes []byte
