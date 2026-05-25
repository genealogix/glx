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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeDirEntry / fakeFileInfo let us drive pluginNameFromEntry deterministically
// on any platform without depending on the host filesystem's mode-bit semantics
// (Windows in particular reports executable-bit oddly for files created via
// os.WriteFile). The mocks implement just the os.DirEntry / fs.FileInfo
// surface that pluginNameFromEntry actually uses.
type fakeDirEntry struct {
	name  string
	mode  fs.FileMode
	isDir bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode          { return f.mode.Type() }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return fakeFileInfo{f.name, f.mode}, nil }

type fakeFileInfo struct {
	name string
	mode fs.FileMode
}

func (fi fakeFileInfo) Name() string       { return fi.name }
func (fi fakeFileInfo) Size() int64        { return 0 }
func (fi fakeFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fi fakeFileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fakeFileInfo) Sys() any           { return nil }

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

// TestDiscoverPlugins_NormalizesRelativePATHEntry verifies that a relative
// PATH directory (e.g., "./bin") is canonicalized to an absolute path at
// discovery time, so the resolved Plugin.Path is stable across any later CWD
// change and exec is given an unambiguous absolute path.
func TestDiscoverPlugins_NormalizesRelativePATHEntry(t *testing.T) {
	parent := t.TempDir()
	relDir := "bin"
	absDir := filepath.Join(parent, relDir)
	if err := os.Mkdir(absDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	makeFakePluginFile(t, absDir, "glx-foo", "echo")
	t.Chdir(parent)

	plugins := discoverPlugins(relDir, defaultPathExtForTest())
	if len(plugins) != 1 {
		t.Fatalf("want 1 plugin from relative PATH entry, got %d: %+v", len(plugins), plugins)
	}
	if !filepath.IsAbs(plugins[0].Path) {
		t.Errorf("Plugin.Path should be normalized to absolute; got %q", plugins[0].Path)
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

// TestDiscoverPlugins_WindowsPATHEXTPrecedence verifies that when the same
// PATH directory holds glx-foo.bat and glx-foo.exe, the .exe wins because it
// appears earlier in PATHEXT — matching how Windows shells resolve names.
func TestDiscoverPlugins_WindowsPATHEXTPrecedence(t *testing.T) {
	if runtime.GOOS != osWindows {
		t.Skip("PATHEXT precedence is a Windows-only concern")
	}
	dir := t.TempDir()
	// Both files have the same logical name "foo"; .exe should win over .bat.
	if err := os.WriteFile(filepath.Join(dir, "glx-foo.bat"), []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatalf("write .bat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "glx-foo.exe"), []byte{0x4d, 0x5a}, 0o644); err != nil {
		t.Fatalf("write .exe: %v", err)
	}
	plugins := discoverPlugins(dir, ".COM;.EXE;.BAT;.CMD")
	if len(plugins) != 1 {
		t.Fatalf("want 1 plugin (single logical name), got %d: %+v", len(plugins), plugins)
	}
	if !strings.HasSuffix(strings.ToLower(plugins[0].Path), ".exe") {
		t.Errorf("PATHEXT precedence: want .exe to win over .bat; got %q", plugins[0].Path)
	}
}

// TestDiscoverPlugins_WindowsUppercaseFilename verifies that the case-insensitive
// prefix match on Windows handles GLX-FOO.BAT just like glx-foo.bat.
func TestDiscoverPlugins_WindowsUppercaseFilename(t *testing.T) {
	if runtime.GOOS != osWindows {
		t.Skip("uppercase prefix matching is a Windows-only concern (NTFS reports case as stored)")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "GLX-FOO.BAT"), []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	plugins := discoverPlugins(dir, ".COM;.EXE;.BAT;.CMD")
	if len(plugins) != 1 || plugins[0].Name != "foo" {
		t.Errorf("want plugin name 'foo' from GLX-FOO.BAT, got %+v", plugins)
	}
}

// TestParsePathExt exercises both isWindows branches deterministically so the
// platform-specific branches are covered by Linux-only CI runs.
func TestParsePathExt(t *testing.T) {
	if got := parsePathExt(".EXE;.BAT", false); got != nil {
		t.Errorf("parsePathExt(_, isWindows=false) = %v; want nil", got)
	}
	got := parsePathExt(".EXE;.BAT", true)
	want := []string{".exe", ".bat"}
	if !slices.Equal(got, want) {
		t.Errorf("parsePathExt(.EXE;.BAT, true) = %v; want %v", got, want)
	}
	// Empty pathExt under isWindows: conservative default kicks in.
	got = parsePathExt("", true)
	want = []string{".com", ".exe", ".bat", ".cmd"}
	if !slices.Equal(got, want) {
		t.Errorf("parsePathExt(empty, true) = %v; want default %v", got, want)
	}
	// Whitespace + empty entries are tolerated.
	got = parsePathExt(" .EXE ; ; .BAT ", true)
	want = []string{".exe", ".bat"}
	if !slices.Equal(got, want) {
		t.Errorf("parsePathExt whitespace handling = %v; want %v", got, want)
	}
}

// TestPluginNameFromEntry exercises both isWindows branches and the various
// rejection paths (directory, non-prefixed, empty stem, missing exec bit)
// without depending on real filesystem mode semantics.
func TestPluginNameFromEntry(t *testing.T) {
	winExts := []string{".com", ".exe", ".bat", ".cmd"}
	tests := []struct {
		name      string
		entry     fakeDirEntry
		exts      []string
		isWindows bool
		wantName  string
		wantIdx   int
		wantOk    bool
	}{
		// --- Windows branch ---
		{"win: glx-foo.exe → foo (idx 1)", fakeDirEntry{name: "glx-foo.exe", mode: 0o644}, winExts, true, "foo", 1, true},
		{"win: GLX-FOO.BAT → foo (case-insensitive prefix + ext)", fakeDirEntry{name: "GLX-FOO.BAT", mode: 0o644}, winExts, true, "foo", 2, true},
		{"win: glx-foo.txt → rejected (not in PATHEXT)", fakeDirEntry{name: "glx-foo.txt", mode: 0o644}, winExts, true, "", 0, false},
		{"win: notglx-foo.exe → rejected (no prefix)", fakeDirEntry{name: "notglx-foo.exe", mode: 0o644}, winExts, true, "", 0, false},
		{"win: glx-.exe → rejected (empty stem)", fakeDirEntry{name: "glx-.exe", mode: 0o644}, winExts, true, "", 0, false},
		{"win: directory glx-foo.exe → rejected", fakeDirEntry{name: "glx-foo.exe", isDir: true, mode: fs.ModeDir | 0o755}, winExts, true, "", 0, false},

		// --- Unix branch ---
		{"unix: glx-foo (exec bit) → foo", fakeDirEntry{name: "glx-foo", mode: 0o755}, nil, false, "foo", 0, true},
		{"unix: glx-foo (no exec bit) → rejected", fakeDirEntry{name: "glx-foo", mode: 0o644}, nil, false, "", 0, false},
		{"unix: glx- (empty stem)", fakeDirEntry{name: "glx-", mode: 0o755}, nil, false, "", 0, false},
		{"unix: notglx-foo → rejected", fakeDirEntry{name: "notglx-foo", mode: 0o755}, nil, false, "", 0, false},
		{"unix: GLX-FOO → rejected (case-sensitive prefix on Unix)", fakeDirEntry{name: "GLX-FOO", mode: 0o755}, nil, false, "", 0, false},
		{"unix: directory glx-foo → rejected", fakeDirEntry{name: "glx-foo", isDir: true, mode: fs.ModeDir | 0o755}, nil, false, "", 0, false},
		{"unix: symlink glx-foo → rejected (non-regular file)", fakeDirEntry{name: "glx-foo", mode: fs.ModeSymlink | 0o777}, nil, false, "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotIdx, gotOk := pluginNameFromEntry(tt.entry, tt.exts, tt.isWindows)
			if gotOk != tt.wantOk || gotName != tt.wantName || gotIdx != tt.wantIdx {
				t.Errorf("pluginNameFromEntry = (%q, %d, %v); want (%q, %d, %v)",
					gotName, gotIdx, gotOk, tt.wantName, tt.wantIdx, tt.wantOk)
			}
		})
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
		{"cobra-internal __complete", []string{"__complete", "validate"}, "__complete", []string{"validate"}, true},
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

func TestQuietFlagRequested(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"nil", nil, false},
		{"-q short", []string{"-q"}, true},
		{"--quiet long", []string{"--quiet"}, true},
		{"--quiet=true", []string{"--quiet=true"}, true},
		{"--quiet=1", []string{"--quiet=1"}, true},
		{"--quiet=false", []string{"--quiet=false"}, false},
		{"--quiet=garbage", []string{"--quiet=garbage"}, false},
		{"-q before --plugins", []string{"-q", "--plugins"}, true},
		{"unrelated", []string{"validate", "/path"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quietFlagRequested(tt.args); got != tt.want {
				t.Errorf("quietFlagRequested(%v) = %v; want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestKnownCommandNames_ContainsBuiltinsAndAutoAdded(t *testing.T) {
	ensureBuiltinSubcommands(rootCmd)
	known := knownCommandNames(rootCmd)
	for _, name := range []string{"validate", "import", "export", "help", "completion"} {
		if !known[name] {
			t.Errorf("knownCommandNames missing %q", name)
		}
	}
}

// TestPluginDispatchTarget covers Execute()'s dispatch decision: built-ins win,
// cobra-internal __complete* falls through, unknown-with-plugin dispatches,
// unknown-without-plugin falls through, and no-subcommand falls through. The
// Windows case-insensitive subset is exercised by the isWindows column so the
// branch is covered on Linux CI.
func TestPluginDispatchTarget(t *testing.T) {
	ensureBuiltinSubcommands(rootCmd)
	dir := t.TempDir()
	// Two fixture plugins: one shadows the built-in `validate` (must not be
	// dispatched), one is a genuine new subcommand (must be dispatched).
	makeFakePluginFile(t, dir, "glx-validate", "echo shadowed")
	makeFakePluginFile(t, dir, "glx-mycmd", "echo mycmd")
	ext := defaultPathExtForTest()

	cases := []struct {
		name      string
		args      []string
		isWindows bool
		wantOk    bool
		wantName  string
		wantRest  []string
	}{
		// Unix-style dispatch: isWindows=false, name lookup is case-sensitive.
		{"unix: built-in beats discovered plugin", []string{"validate", "/some/path"}, false, false, "", nil},
		{"unix: unknown name with no plugin", []string{"definitely-not-a-plugin"}, false, false, "", nil},
		{"unix: unknown name with plugin → dispatch", []string{"mycmd", "--x", "y"}, false, true, "mycmd", []string{"--x", "y"}},
		{"cobra __complete falls through", []string{"__complete", "validate"}, false, false, "", nil},
		{"no subcommand (only flags) falls through", []string{"--plugins"}, false, false, "", nil},
		{"empty args falls through", []string{}, false, false, "", nil},

		// Windows-style dispatch: isWindows=true, lookup is lowercased.
		// Fixture plugins on disk are named glx-mycmd (lowercase), so the
		// lowercase-normalized lookup matches on any host filesystem.
		{"win: uppercase name dispatches to lowercase plugin", []string{"MYCMD", "a"}, true, true, "mycmd", []string{"a"}},
		{"win: mixed-case name dispatches", []string{"Mycmd"}, true, true, "mycmd", []string{}},
		{"win: uppercase built-in still wins (no plugin shadow)", []string{"VALIDATE"}, true, false, "", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			p, rest, ok := pluginDispatchTarget(rootCmd, tt.args, dir, ext, tt.isWindows)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v; want %v (plugin=%+v rest=%v)", ok, tt.wantOk, p, rest)
			}
			if !ok {
				return
			}
			if p.Name != tt.wantName {
				t.Errorf("plugin.Name = %q; want %q", p.Name, tt.wantName)
			}
			if !slices.Equal(rest, tt.wantRest) {
				t.Errorf("rest = %v; want %v", rest, tt.wantRest)
			}
		})
	}
}

func TestListPlugins_EmptyNotice(t *testing.T) {
	var buf bytes.Buffer
	listPlugins(nil, map[string]bool{}, false /* quiet */, &buf)
	if !strings.Contains(buf.String(), "No glx plugins found") {
		t.Errorf("empty listing should explain how to install plugins; got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "glx-<name>") {
		t.Errorf("empty listing should describe the naming convention; got %q", buf.String())
	}
}

// TestListPlugins_EmptyUnderQuietSuppressesNotice — `--quiet` suppresses the
// informational "no plugins found" notice (it's a warning-like message), per
// the established quiet-output contract.
func TestListPlugins_EmptyUnderQuietSuppressesNotice(t *testing.T) {
	var buf bytes.Buffer
	listPlugins(nil, map[string]bool{}, true /* quiet */, &buf)
	if buf.Len() != 0 {
		t.Errorf("under --quiet, empty listing should suppress the notice; got %q", buf.String())
	}
}

// TestListPlugins_NonEmptyUnderQuietStillPrints — `--quiet` does NOT suppress
// the actual listing, because the listing IS the user-requested output (the
// answer to `--plugins`). This mirrors `glx add`'s rule that the entity ID is
// emitted on IOStreams.MachineOut regardless of quiet.
func TestListPlugins_NonEmptyUnderQuietStillPrints(t *testing.T) {
	plugins := []Plugin{{Name: "foo", Path: "/x/glx-foo"}}
	var buf bytes.Buffer
	listPlugins(plugins, map[string]bool{}, true /* quiet */, &buf)
	if !strings.Contains(buf.String(), "foo") || !strings.Contains(buf.String(), "/x/glx-foo") {
		t.Errorf("listing must still print under --quiet (it's the requested output); got %q", buf.String())
	}
}

func TestListPlugins_FormattedWithShadowedAnnotation(t *testing.T) {
	plugins := []Plugin{
		{Name: "foo", Path: "/x/glx-foo"},
		{Name: "validate", Path: "/x/glx-validate"},
	}
	var buf bytes.Buffer
	listPlugins(plugins, map[string]bool{"validate": true}, false /* quiet */, &buf)
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

// TestRunPlugin_SuccessExit exercises the exit-code=0 happy path via the same
// helper-process pattern.
func TestRunPlugin_SuccessExit(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	p := Plugin{Name: "helper", Path: exe}
	args := []string{"-test.run=TestHelperProcess", "--", "echo-and-exit", "0"}
	var stdout, stderr bytes.Buffer
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	code := runPlugin(context.Background(), p, args, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Errorf("want exit 0 on success path, got %d (stderr=%q)", code, stderr.String())
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
