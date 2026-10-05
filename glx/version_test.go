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
	"runtime/debug"
	"strings"
	"testing"
)

// fakeBuildInfo builds a BuildInfo as Go records it: a main-module version
// plus key/value build settings (keys and values alternate in kv).
func fakeBuildInfo(mainVersion string, kv ...string) *debug.BuildInfo {
	info := &debug.BuildInfo{Main: debug.Module{Path: "github.com/genealogix/glx", Version: mainVersion}}
	for i := 0; i+1 < len(kv); i += 2 {
		info.Settings = append(info.Settings, debug.BuildSetting{Key: kv[i], Value: kv[i+1]})
	}

	return info
}

const (
	testRevision = "ea402a528f36bce5c04fcdb19a842a671a80d889"
	testVCSTime  = "2026-10-01T14:46:10Z"
	testPseudo   = "v0.0.0-beta.12.0.20261001144610-ea402a528f36"
)

func TestFormatVersion(t *testing.T) {
	// A source checkout's stamp: Go 1.24+ also derives a pseudo-version for it.
	sourceBuild := fakeBuildInfo(testPseudo,
		"vcs", "git", "vcs.revision", testRevision, "vcs.time", testVCSTime, "vcs.modified", "false")

	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		info    *debug.BuildInfo
		want    string
	}{
		// Injected via -ldflags (GoReleaser, Makefile): output is unchanged and
		// the build info is ignored.
		{"version only", "1.0.0", "", "", nil, "1.0.0"},
		{"version+commit", "1.0.0", "abc1234def5678", "", nil, "1.0.0 (abc1234)"},
		{"version+commit+date", "1.0.0", "abc1234def5678", "2026-03-30", nil, "1.0.0 (abc1234) 2026-03-30"},
		{"short commit", "1.0.0", "abc", "", nil, "1.0.0 (abc)"},
		{"injected version ignores build info", "1.0.0", "", "", sourceBuild, "1.0.0"},
		{"injected commit ignores build info", "dev", "abc1234def5678", "", sourceBuild, "dev (abc1234)"},
		{"dev without build info", "dev", "", "", nil, "dev"},

		// Plain go build / go install fallbacks (#1337).
		{"dev, no stamp", "dev", "", "", fakeBuildInfo("(devel)"), "dev"},
		{"source checkout", "dev", "", "", sourceBuild, "dev (ea402a5, " + testVCSTime + ")"},
		{
			"source checkout, uncommitted changes", "dev", "", "",
			fakeBuildInfo(testPseudo+"+dirty", "vcs.revision", testRevision, "vcs.time", testVCSTime, "vcs.modified", "true"),
			"dev (ea402a5-dirty, " + testVCSTime + ")",
		},
		{
			"source checkout, pre-1.24 (devel)", "dev", "", "",
			fakeBuildInfo("(devel)", "vcs.revision", testRevision, "vcs.time", testVCSTime),
			"dev (ea402a5, " + testVCSTime + ")",
		},
		{
			"tagged clean checkout", "dev", "", "",
			fakeBuildInfo("v0.0.0-beta.12", "vcs.revision", testRevision, "vcs.time", testVCSTime, "vcs.modified", "false"),
			"v0.0.0-beta.12 (ea402a5, " + testVCSTime + ")",
		},
		{"go install @tag", "dev", "", "", fakeBuildInfo("v0.0.0-beta.12"), "v0.0.0-beta.12"},
		{"go install @main", "dev", "", "", fakeBuildInfo(testPseudo), testPseudo},
		{"revision without time", "dev", "", "", fakeBuildInfo("", "vcs.revision", "abc"), "dev (abc)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatVersion(tt.version, tt.commit, tt.date, tt.info); got != tt.want {
				t.Errorf("formatVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// versionString reads the test binary's own build info; whatever it holds,
// injected values still take precedence and a dev build never comes out empty.
func TestVersionString(t *testing.T) {
	origVersion, origCommit, origDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = origVersion, origCommit, origDate
	})

	version, commit, date = "1.0.0", "abc1234def5678", "2026-03-30"
	if got, want := versionString(), "1.0.0 (abc1234) 2026-03-30"; got != want {
		t.Errorf("versionString() = %q, want %q", got, want)
	}

	version, commit, date = "dev", "", ""
	if got := versionString(); got == "" || strings.HasPrefix(got, " ") {
		t.Errorf("versionString() = %q, want a non-empty version", got)
	}
}
