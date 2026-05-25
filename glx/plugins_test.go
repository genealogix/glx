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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// makeFakePluginFile writes an executable file at dir/base that discoverPlugins
// will recognize on the current platform. On Windows it writes <base>.bat; on
// Unix it writes <base> with the executable bit set.
func makeFakePluginFile(t *testing.T, dir, base, body string) {
	t.Helper()
	if runtime.GOOS == osWindows {
		path := filepath.Join(dir, base+".bat")
		if err := os.WriteFile(path, []byte("@echo off\r\n"+body+"\r\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}

		return
	}
	path := filepath.Join(dir, base)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func defaultPathExtForTest() string {
	if runtime.GOOS == osWindows {
		return ".COM;.EXE;.BAT;.CMD"
	}

	return ""
}

func TestDiscoverPlugins_FindsAndSortsAndDedups(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	makeFakePluginFile(t, dir1, "glx-foo", "echo foo")
	makeFakePluginFile(t, dir1, "glx-bar", "echo bar")

	// Non-matching file should be ignored.
	if err := os.WriteFile(filepath.Join(dir1, "other"), []byte("x"), 0o755); err != nil {
		t.Fatalf("write other: %v", err)
	}
	// A subdirectory starting with glx- should be skipped (not a regular file).
	if err := os.Mkdir(filepath.Join(dir1, "glx-baz"), 0o755); err != nil {
		t.Fatalf("mkdir glx-baz: %v", err)
	}
	// Duplicate of glx-foo in dir2 (later on PATH): first occurrence wins.
	makeFakePluginFile(t, dir2, "glx-foo", "echo foo-shadowed")

	pathEnv := dir1 + string(os.PathListSeparator) + dir2
	plugins := discoverPlugins(pathEnv, defaultPathExtForTest())

	if len(plugins) != 2 {
		t.Fatalf("want 2 plugins, got %d: %+v", len(plugins), plugins)
	}
	if plugins[0].Name != "bar" || plugins[1].Name != "foo" {
		t.Errorf("want sorted [bar, foo], got [%q, %q]", plugins[0].Name, plugins[1].Name)
	}
	if !strings.HasPrefix(plugins[1].Path, dir1) {
		t.Errorf("dedup: foo should resolve to dir1 (first on PATH); got %q", plugins[1].Path)
	}
}

func TestDiscoverPlugins_SkipsUnreadableAndEmptyEntries(t *testing.T) {
	sep := string(os.PathListSeparator)
	pathEnv := sep + "  " + sep + filepath.Join(t.TempDir(), "does-not-exist")
	plugins := discoverPlugins(pathEnv, defaultPathExtForTest())
	if len(plugins) != 0 {
		t.Errorf("want 0 plugins from empty/nonexistent PATH entries, got %+v", plugins)
	}
}

func TestFindPlugin_HitAndMiss(t *testing.T) {
	dir := t.TempDir()
	makeFakePluginFile(t, dir, "glx-helloworld", "echo hi")
	p, ok := findPlugin("helloworld", dir, defaultPathExtForTest())
	if !ok {
		t.Fatalf("findPlugin(helloworld) ok=false; want true")
	}
	if p.Name != "helloworld" || filepath.Dir(p.Path) != dir {
		t.Errorf("unexpected plugin: %+v", p)
	}
	if _, ok := findPlugin("missing", dir, defaultPathExtForTest()); ok {
		t.Errorf("findPlugin(missing) ok=true; want false")
	}
}

func TestFirstSubcommandToken(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantName string
		wantRest []string
		wantOk   bool
	}{
		{"nil", nil, "", nil, false},
		{"empty", []string{}, "", nil, false},
		{"only --plugins", []string{"--plugins"}, "", nil, false},
		{"only --help", []string{"--help"}, "", nil, false},
		{"single sub", []string{"foo"}, "foo", []string{}, true},
		{"sub with flags", []string{"foo", "--x", "y"}, "foo", []string{"--x", "y"}, true},
		{"global flag then sub", []string{"-q", "foo", "bar"}, "foo", []string{"bar"}, true},
		{"long global then sub", []string{"--quiet", "validate"}, "validate", []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotRest, gotOk := firstSubcommandToken(tt.args)
			if gotOk != tt.wantOk || gotName != tt.wantName || !slices.Equal(gotRest, tt.wantRest) {
				t.Errorf("firstSubcommandToken(%v) = (%q, %v, %v); want (%q, %v, %v)",
					tt.args, gotName, gotRest, gotOk, tt.wantName, tt.wantRest, tt.wantOk)
			}
		})
	}
}

func TestPluginsFlagRequested(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"nil", nil, false},
		{"bare --plugins", []string{"--plugins"}, true},
		{"--plugins after global flag", []string{"-q", "--plugins"}, true},
		{"--plugins=true (value-attached truthy)", []string{"--plugins=true"}, true},
		{"--plugins=1 (value-attached truthy)", []string{"--plugins=1"}, true},
		{"--plugins=false (value-attached falsey)", []string{"--plugins=false"}, false},
		{"--plugins=garbage (unparseable value)", []string{"--plugins=garbage"}, false},
		{"--plugins before a subcommand → subcommand wins", []string{"--plugins", "foo"}, false},
		{"--plugins after a subcommand", []string{"foo", "--plugins"}, false},
		{"unrelated invocation", []string{"validate", "/some/dir"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pluginsFlagRequested(tt.args); got != tt.want {
				t.Errorf("pluginsFlagRequested(%v) = %v; want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestKnownCommandNames_ContainsBuiltinsAndAutoAdded(t *testing.T) {
	known := knownCommandNames(rootCmd)
	for _, name := range []string{"validate", "import", "export", "help", "completion"} {
		if !known[name] {
			t.Errorf("knownCommandNames missing %q", name)
		}
	}
}

func TestListPlugins_EmptyNotice(t *testing.T) {
	var buf bytes.Buffer
	listPlugins(nil, map[string]bool{}, &buf)
	if !strings.Contains(buf.String(), "No glx plugins found") {
		t.Errorf("empty listing should explain how to install plugins; got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "glx-<name>") {
		t.Errorf("empty listing should describe the naming convention; got %q", buf.String())
	}
}

func TestListPlugins_FormattedWithShadowedAnnotation(t *testing.T) {
	plugins := []Plugin{
		{Name: "foo", Path: "/x/glx-foo"},
		{Name: "validate", Path: "/x/glx-validate"},
	}
	var buf bytes.Buffer
	listPlugins(plugins, map[string]bool{"validate": true}, &buf)
	out := buf.String()

	if !strings.Contains(out, "NAME") || !strings.Contains(out, "PATH") {
		t.Errorf("listing should include a header row; got %q", out)
	}
	if !strings.Contains(out, "foo") || !strings.Contains(out, "/x/glx-foo") {
		t.Errorf("listing missing foo entry: %q", out)
	}
	if !strings.Contains(out, "validate") || !strings.Contains(out, "shadowed by built-in command") {
		t.Errorf("shadowed plugin should be annotated; got %q", out)
	}
}

// TestRunPlugin_ExitCodeAndStdio uses the stdlib os/exec helper-process pattern:
// the test binary is re-invoked as the "plugin" so the test runs hermetically
// and identically on Windows and Unix without external shell or .bat fixtures.
func TestRunPlugin_ExitCodeAndStdio(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	p := Plugin{Name: "helper", Path: exe}
	args := []string{"-test.run=TestHelperProcess", "--", "echo-and-exit", "7"}

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("ping\n")

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	code := runPlugin(context.Background(), p, args, stdin, &stdout, &stderr)

	if code != 7 {
		t.Errorf("want exit 7, got %d (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ping") {
		t.Errorf("stdin → stdout pipe not honored: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "stderr-marker-from-helper") {
		t.Errorf("stderr forwarding failed; got %q", stderr.String())
	}
}

func TestRunPlugin_StartFailureReturns127(t *testing.T) {
	p := Plugin{Name: "no-such", Path: filepath.Join(t.TempDir(), "no-such-binary")}
	var stdout, stderr bytes.Buffer
	code := runPlugin(context.Background(), p, nil, nil, &stdout, &stderr)
	if code != exitPluginStartFailure {
		t.Errorf("want exit %d on start failure, got %d", exitPluginStartFailure, code)
	}
	if !strings.Contains(stderr.String(), "failed to run plugin") {
		t.Errorf("start-failure diagnostic missing on stderr: %q", stderr.String())
	}
}

// TestHelperProcess is not a real test — it is the body executed when this
// binary is re-invoked as a fake plugin via the os/exec helper-process pattern.
// It does nothing when GO_WANT_HELPER_PROCESS is unset, so normal test runs
// pass through it unaffected. The *testing.T parameter is required by the
// `go test` signature for Test* functions but is intentionally unused.
func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]

			break
		}
	}
	if len(args) == 0 {
		os.Exit(99)
	}
	if args[0] == "echo-and-exit" {
		// Behavior: copy stdin → stdout, write a marker to stderr, exit with <code>.
		if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, "helper copy err:", err)
		}
		fmt.Fprintln(os.Stderr, "stderr-marker-from-helper")
		var code int
		if len(args) >= 2 {
			_, _ = fmt.Sscanf(args[1], "%d", &code)
		}
		os.Exit(code)
	}
	os.Exit(99)
}
