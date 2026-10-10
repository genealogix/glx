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

package e2e

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoverage_WitnessAssertionDoesNotInheritGroomRole(t *testing.T) {
	work := t.TempDir()
	archive := `event_types:
  marriage: {label: Marriage}
persons:
  john: {properties: {name: John}}
  jane: {properties: {name: Jane}}
events:
  marriage:
    type: marriage
    participants: [{person: john, role: groom}, {person: jane, role: bride}]
sources:
  register: {title: Parish register, type: church_register}
citations:
  witness: {source: register}
assertions:
  john-witness:
    subject: {event: marriage}
    participant: {person: john, role: witness}
    citations: [witness]
`
	require.NoError(t, os.WriteFile(filepath.Join(work, "archive.glx"), []byte(archive), 0o644))
	require.Equal(t, 0, runGLX(t, work, "validate", "archive.glx").exitCode)
	res := runGLX(t, work, "coverage", "john", "--archive", "archive.glx", "--json")
	require.Equal(t, 0, res.exitCode, res.stderr)
	var result struct {
		Records []struct {
			Label string `json:"label"`
			Found bool   `json:"found"`
		} `json:"records"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.stdout), &result))
	for _, record := range result.Records {
		if record.Label == "Church records" {
			assert.False(t, record.Found, "the source asserts a witness role, not the groom conclusion")

			return
		}
	}
	t.Fatal("Church records row missing")
}

func TestPublish_LivingDropsEventCitationTextAndMedia(t *testing.T) {
	work := t.TempDir()
	archive := `event_types:
  birth: {label: Birth}
  marriage: {label: Marriage}
persons:
  private: {properties: {name: Alex Private, living: true}}
  public: {properties: {name: Old Public, living: false}}
events:
  birth-private:
    type: birth
    date: '2000-05-12'
    participants: [{person: private, role: principal}]
  birth-public:
    type: birth
    date: '1800'
    participants: [{person: public, role: principal}]
  marriage:
    type: marriage
    participants: [{person: private, role: spouse}, {person: public, role: spouse}]
relationships:
  couple:
    type: marriage
    start_event: marriage
    participants: [{person: private, role: spouse}, {person: public, role: spouse}]
sources:
  register: {title: Register}
citations:
  private-entry:
    source: register
    properties: {text_from_source: 'PRIVATE_QUOTE Alex Private born 2000-05-12 at 7 Secret Street'}
    media: [private-image]
  public-entry:
    source: register
    properties: {text_from_source: PUBLIC_QUOTE Old Public}
    media: [public-image]
media:
  private-image: {uri: private.png, mime_type: image/png, title: PRIVATE_IMAGE}
  public-image: {uri: public.png, mime_type: image/png, title: PUBLIC_IMAGE}
assertions:
  private-birth:
    subject: {event: birth-private}
    property: date
    value: '2000-05-12'
    citations: [private-entry]
  private-marriage:
    subject: {event: marriage}
    citations: [private-entry]
  public-birth:
    subject: {event: birth-public}
    property: date
    value: '1800'
    citations: [public-entry]
`
	privateBytes, publicBytes := []byte("PRIVATE_MEDIA_BYTES"), []byte("PUBLIC_MEDIA_BYTES")
	require.NoError(t, os.WriteFile(filepath.Join(work, "archive.glx"), []byte(archive), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(work, "private.png"), privateBytes, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(work, "public.png"), publicBytes, 0o644))
	require.Equal(t, 0, runGLX(t, work, "validate", "archive.glx").exitCode)
	siteText := func(out string) string {
		t.Helper()
		var all strings.Builder
		for _, data := range snapshotTree(t, out) {
			all.Write(data)
		}

		return all.String()
	}
	plain := filepath.Join(t.TempDir(), "site")
	res := runGLX(t, work, "publish", "--archive", "archive.glx", "--output", plain)
	require.Equal(t, 0, res.exitCode, res.stderr)
	require.Contains(t, siteText(plain), "PRIVATE_QUOTE")
	require.Contains(t, siteText(plain), string(privateBytes))

	for _, embed := range []bool{false, true} {
		t.Run(map[bool]string{false: "copied", true: "embedded"}[embed], func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "site")
			args := []string{"publish", "--archive", "archive.glx", "--output", out, "--living"}
			if embed {
				args = append(args, "--embed-media")
			}
			res := runGLX(t, work, args...)
			require.Equal(t, 0, res.exitCode, res.stderr)
			text := siteText(out)
			for _, private := range []string{"PRIVATE_QUOTE", "Alex Private", "2000-05-12", "7 Secret Street", "PRIVATE_IMAGE", string(privateBytes), base64.StdEncoding.EncodeToString(privateBytes)} {
				assert.NotContains(t, text, private)
			}
			assert.Contains(t, text, "PUBLIC_QUOTE")
			assert.Contains(t, text, "PUBLIC_IMAGE")
			publicImage := string(publicBytes)
			if embed {
				publicImage = base64.StdEncoding.EncodeToString(publicBytes)
			}
			assert.Contains(t, text, publicImage, "unredacted evidence must remain published")
		})
	}
}
