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
	"regexp"
	"runtime/debug"
	"strings"
)

// devVersion is the value of main.version when no -ldflags injected one.
const devVersion = "dev"

// shortRevisionLen is how many characters of a commit hash are displayed.
const shortRevisionLen = 7

// pseudoVersionSuffix matches the "yyyymmddhhmmss-abcdefabcdef" tail (with an
// optional "+dirty") of a Go pseudo-version such as
// v0.0.0-beta.12.0.20261001144610-ea402a528f36.
var pseudoVersionSuffix = regexp.MustCompile(`[.-]\d{14}-[0-9a-f]{12}(\+dirty)?$`)

// versionString is the text `glx --version` prints after "glx version ".
func versionString() string {
	info, _ := debug.ReadBuildInfo()

	return formatVersion(version, commit, date, info)
}

// formatVersion renders the version line from the ldflags-injected values and,
// for builds that injected nothing (plain `go build` / `go install`), from the
// module version and VCS stamp Go records in every binary (#1337).
//
// Injected builds (GoReleaser sets version, commit and date; the Makefile sets
// version) keep their established form: "1.2.3 (abc1234) 2026-03-30". For a
// "dev" build without an injected commit:
//
//   - info.Main.Version replaces "dev" when it names a real version, e.g.
//     v0.0.0-beta.12 from `go install ...@v0.0.0-beta.12`. A pseudo-version is
//     only used when there is no VCS stamp to show instead, since Go derives it
//     from the same revision and time.
//   - vcs.revision (shortened), "-dirty" when vcs.modified is true, and
//     vcs.time are appended in parentheses: "dev (ea402a5, 2026-10-01T14:46:10Z)".
func formatVersion(ver, commitHash, buildDate string, info *debug.BuildInfo) string {
	if ver != devVersion || commitHash != "" || info == nil {
		return injectedVersion(ver, commitHash, buildDate)
	}

	settings := make(map[string]string, len(info.Settings))
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	revision := settings["vcs.revision"]

	v := ver
	if mv := info.Main.Version; mv != "" && mv != "(devel)" &&
		(revision == "" || !pseudoVersionSuffix.MatchString(mv)) {
		v = mv
	}

	var details []string
	if revision != "" {
		rev := revision[:min(len(revision), shortRevisionLen)]
		if settings["vcs.modified"] == "true" {
			rev += "-dirty"
		}
		details = append(details, rev)
	}
	if t := settings["vcs.time"]; t != "" {
		details = append(details, t)
	}
	if len(details) > 0 {
		v += " (" + strings.Join(details, ", ") + ")"
	}

	return v
}

// injectedVersion formats the values set via -ldflags.
func injectedVersion(ver, commitHash, buildDate string) string {
	v := ver
	if commitHash != "" {
		v += " (" + commitHash[:min(len(commitHash), shortRevisionLen)] + ")"
	}
	if buildDate != "" {
		v += " " + buildDate
	}

	return v
}
