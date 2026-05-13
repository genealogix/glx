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

package glx

import (
	"fmt"
	"strings"
)

// windowsReservedNames lists Windows device names that cannot be used as
// filenames, even with an extension (e.g., "CON.glx" is invalid).
var windowsReservedNames = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// EntityIDToFilename derives a deterministic filename from an entity ID.
// The entity ID is lowercased to reduce case-insensitive filesystem collisions
// (e.g., on Windows/macOS where "Person-A.glx" and "person-a.glx" would collide).
// Returns an error if the entity ID contains path separators, dot-segments,
// Windows reserved device names, or other characters unsafe for use as a filename.
func EntityIDToFilename(entityID string) (string, error) {
	if entityID == "" ||
		strings.ContainsAny(entityID, "/\\:") ||
		entityID == "." || entityID == ".." ||
		strings.HasPrefix(entityID, ".") {
		return "", fmt.Errorf("%w: %q", ErrUnsafeEntityID, entityID)
	}

	lower := strings.ToLower(entityID)
	if windowsReservedNames[lower] {
		return "", fmt.Errorf("%w: %q (Windows reserved device name)", ErrUnsafeEntityID, entityID)
	}

	return lower + ".glx", nil
}
