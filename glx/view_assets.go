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
	"embed"
)

// viewTemplatesFS holds the HTML templates compiled into the binary so
// `glx view` has no runtime template dependencies.
//
//go:embed view/templates/*.tmpl
var viewTemplatesFS embed.FS

// viewStaticAssets maps an embedded asset's base name to its output path,
// relative to the generated site root. Order is irrelevant; the keys are the
// embedded files under view/assets.
var viewStaticAssets = map[string]string{
	"style.css": "css/style.css",
	"search.js": "js/search.js",
	"theme.js":  "js/theme.js",
}

// viewAssetsFS holds the CSS and JS assets copied verbatim into the site.
//
//go:embed view/assets/*
var viewAssetsFS embed.FS
