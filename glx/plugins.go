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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// pluginPrefix is the naming convention for discoverable plugins. Any executable
// named glx-<name> on the user's PATH is exposed as `glx <name>`, mirroring the
// git/kubectl plugin model. This is Phase 1 of the plugin/extension system
// proposed in issue #95: discovery and exec-based dispatch only — no archive
// data is passed to the plugin, no manifest is read, and no protocol is defined.
// Those are deferred to follow-up issue phases (validators, importers/exporters,
// hooks, SDK, registry).
const pluginPrefix = "glx-"

// osWindows is the runtime.GOOS value identifying Windows builds. Hoisted to a
// named constant to satisfy the goconst linter and keep platform branches easy
// to find.
const osWindows = "windows"

// Flag names and the quiet shorthand shared by Cobra and the prefix parser.
const (
	rootFlagPlugins = "plugins"
	rootFlagQuiet   = "quiet"
	quietShorthand  = "q"
)

// exitPluginStartFailure is the exit code returned when the plugin executable
// cannot be started at all (vs. starting and exiting non-zero). 127 matches the
// shell convention for "command not found / not executable".
const exitPluginStartFailure = 127

// Plugin is an executable discovered on PATH that handles `glx <Name>` invocations.
type Plugin struct {
	// Name is the plugin's subcommand name: the filename with the glx- prefix and,
	// on Windows, the executable extension stripped (e.g. glx-german-places.exe →
	// "german-places").
	Name string

	// Path is the absolute path to the executable that will be exec'd when
	// `glx <Name>` is invoked. discoverPlugins normalizes each PATH directory
	// to absolute form via filepath.Abs before joining the filename, so this
	// is always absolute and contains a separator — exec.CommandContext does
	// NOT route through exec.LookPath/PATH, and Go 1.19+ ErrDot/CWD-resolution
	// ambiguity does not apply even when PATH had relative entries.
	//
	// When the PATH entry is a symlink (a package-manager shim), this is the
	// link's own path, not the resolved target: exec follows the link itself,
	// and reporting the PATH location is what `glx --plugins` should show and
	// what the user can act on. Discovery still validates the link target — see
	// pluginNameFromEntry.
	Path string
}

// discoverPlugins scans the directories in pathEnv (the PATH environment variable)
// for executables named glx-<name> and returns them sorted by Name. When the same
// plugin name appears in more than one PATH directory the first occurrence wins,
// matching git's plugin resolution. When the same PATH directory contains the
// same plugin name under multiple Windows PATHEXT extensions (e.g. glx-foo.exe
// and glx-foo.bat), the one whose extension appears earliest in PATHEXT wins,
// matching how Windows shells resolve command names.
//
// pathExt is consulted on Windows to determine which file extensions count as
// executable (PATHEXT, default ".COM;.EXE;.BAT;.CMD"); it is ignored on other
// platforms, where the executable bit is checked instead. Directories are
// skipped. Unreadable or non-existent PATH directories are skipped silently,
// mirroring shell behavior.
func discoverPlugins(pathEnv, pathExt string) []Plugin {
	isWindows := runtime.GOOS == osWindows
	exts := parsePathExt(pathExt, isWindows)
	seen := make(map[string]struct{})
	var plugins []Plugin
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		// Do NOT trim whitespace from dir: on Unix, leading/trailing spaces
		// are valid in directory names (e.g., "/opt/tools "), so trimming
		// would render legitimate PATH entries undiscoverable. An entry that
		// is *only* whitespace will fall through harmlessly — filepath.Abs
		// resolves it relative to CWD and os.ReadDir below will fail, so it
		// is skipped without changing observable behavior.
		//
		// Normalize relative PATH entries (e.g., ".", "./bin") to absolute
		// paths at discovery time so the resolved Plugin.Path is stable
		// regardless of any later CWD change, and so exec.CommandContext
		// receives an absolute path that cannot be misinterpreted relative
		// to CWD. Skip the entry on failure (rare — only when CWD itself
		// cannot be determined).
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		dir = absDir
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		// Within this directory, group entries by logical plugin name and keep
		// the one whose extension has the highest PATHEXT priority (lowest index).
		// On non-Windows extIdx is always 0, so the first match wins, which is fine.
		type cand struct {
			extIdx int
			fn     string
		}
		best := make(map[string]cand)
		for _, e := range entries {
			name, extIdx, ok := pluginNameFromEntry(dir, e, exts, isWindows)
			if !ok {
				continue
			}
			if cur, exists := best[name]; !exists || extIdx < cur.extIdx {
				best[name] = cand{extIdx: extIdx, fn: e.Name()}
			}
		}
		// Materialize this directory's winners, honoring cross-directory dedup
		// (first PATH dir wins). Sort the per-dir keys so the result is stable
		// across runs even though map iteration is randomized.
		dirNames := make([]string, 0, len(best))
		for n := range best {
			dirNames = append(dirNames, n)
		}
		sort.Strings(dirNames)
		for _, name := range dirNames {
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			plugins = append(plugins, Plugin{Name: name, Path: filepath.Join(dir, best[name].fn)})
		}
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Name < plugins[j].Name })

	return plugins
}

// findPlugin resolves the plugin for the given subcommand name, if any, with
// the same answer discoverPlugins would give for that name — first PATH
// directory wins, and within a directory the earliest PATHEXT extension wins —
// but without listing any directory. The dispatch path (`glx <name>`) needs
// exactly one candidate, so it stats `glx-<name>` (on Windows,
// `glx-<name><ext>` for each PATHEXT extension in order) in each PATH
// directory instead of reading every entry of every directory, which on a
// typical PATH means walking /usr/bin and friends on every invocation.
// discoverPlugins remains the implementation behind `glx --plugins`.
//
// Because the name is joined onto a directory rather than matched against a
// listing, a name that is not a single path component (one containing a
// separator, or on Windows a drive/stream colon) is rejected up front: a listing
// could never have produced it, and joining it would reach files outside the
// PATH directory itself.
func findPlugin(name, pathEnv, pathExt string) (Plugin, bool) {
	isWindows := runtime.GOOS == osWindows
	if !isPluginLookupName(name, isWindows) {
		return Plugin{}, false
	}
	candidates := []string{pluginPrefix + name}
	if isWindows {
		exts := parsePathExt(pathExt, isWindows)
		candidates = make([]string, 0, len(exts))
		for _, ext := range exts {
			candidates = append(candidates, pluginPrefix+name+ext)
		}
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		// Same normalization as discoverPlugins: an absolute Plugin.Path that
		// does not depend on CWD at exec time.
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		for _, fn := range candidates {
			path := filepath.Join(absDir, fn)
			// os.Stat follows symlinks, so a PATH shim qualifies when its target
			// is a regular executable file, exactly as in pluginNameFromEntry.
			// #nosec G703 -- path is an absolute PATH dir joined with a single
			// file name that isPluginLookupName has already confined (no
			// separators, NUL, drive or ADS syntax); Stat only reads metadata.
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if !isWindows && info.Mode()&0o111 == 0 {
				continue
			}

			return Plugin{Name: name, Path: path}, true
		}
	}

	return Plugin{}, false
}

// isPluginLookupName reports whether name can be looked up as a single file
// name `glx-<name>` inside a PATH directory: non-empty, no NUL, no path
// separator, and on Windows no `\` or `:` (which would address a drive or an
// NTFS alternate data stream).
func isPluginLookupName(name string, isWindows bool) bool {
	if name == "" || strings.ContainsAny(name, "/\x00") {
		return false
	}
	if isWindows && strings.ContainsAny(name, `\:`) {
		return false
	}

	return !strings.ContainsRune(name, filepath.Separator)
}

// pluginNameFromEntry extracts a plugin's logical name from an entry in the PATH
// directory dir. Returns ok=false unless the entry resolves to a regular file
// with the glx- prefix that is executable on the target platform. The
// regular-file requirement is enforced on both platform branches: directories,
// devices, sockets, and FIFOs are never exec'd as plugins, so such an entry
// falls through to cobra's normal "unknown command" error rather than producing
// a plugin start failure.
//
// Symlinks are resolved and their target is what must satisfy the
// regular-file/executable test. PATH shims (e.g. /usr/local/bin/glx-foo →
// /opt/foo/bin/glx-foo) are how package managers such as Homebrew, npm, and asdf
// install executables, and git/kubectl dispatch through them, so rejecting
// symlinks outright would make the most common installation style
// undiscoverable. Requiring the *target* to be a regular file keeps
// symlinks-to-directories, dangling links, and link cycles rejected. dir is
// needed to perform that resolution because os.DirEntry carries only a base name.
//
// When isWindows is true, the entry's extension must be in exts; the returned
// extIdx is the matched extension's index in exts (lower = higher PATHEXT
// priority) and is used by discoverPlugins to pick the right file when several
// extensions co-exist. PATHEXT membership is what makes a file executable on
// Windows, so no permission bit is consulted there. Otherwise the file must have
// at least one executable permission bit set; extIdx is always 0.
//
// isWindows is taken as a parameter (rather than read from runtime.GOOS inside
// the function) so both platform branches are exercised by unit tests on any
// host.
func pluginNameFromEntry(dir string, e os.DirEntry, exts []string, isWindows bool) (name string, extIdx int, ok bool) {
	if e.IsDir() {
		return "", 0, false
	}
	fn := e.Name()
	matchName := fn
	if isWindows {
		matchName = strings.ToLower(fn)
	}
	if !strings.HasPrefix(matchName, pluginPrefix) {
		return "", 0, false
	}
	info, isFile := resolveRegularFile(dir, e)
	if !isFile {
		return "", 0, false
	}
	if isWindows {
		for i, ext := range exts {
			if strings.HasSuffix(matchName, ext) {
				stem := strings.TrimSuffix(strings.TrimPrefix(matchName, pluginPrefix), ext)
				if stem == "" {
					return "", 0, false
				}

				return stem, i, true
			}
		}

		return "", 0, false
	}
	if info.Mode()&0o111 == 0 {
		return "", 0, false
	}
	stem := strings.TrimPrefix(fn, pluginPrefix)
	if stem == "" {
		return "", 0, false
	}

	return stem, 0, true
}

// resolveRegularFile returns the FileInfo that governs whether e (an entry in
// dir) may be exec'd as a plugin, following symlinks to do so.
//
// Whether os.DirEntry.Info follows symlinks is implementation-dependent, so
// this does not rely on it either way: whenever the entry's info still reports
// a symlink, the joined path is os.Stat'd to inspect what it points at, and
// when Info has already resolved the link that step is simply skipped. os.Stat
// walks the whole link chain and returns an error for dangling links and
// cycles, so both are rejected here. Anything that is not ultimately a
// regular file — a directory, device, socket, or FIFO, whether named directly or
// reached through a link — is rejected as well.
func resolveRegularFile(dir string, e os.DirEntry) (fs.FileInfo, bool) {
	info, err := e.Info()
	if err != nil {
		return nil, false
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Stat(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, false
		}
		info = target
	}
	if !info.Mode().IsRegular() {
		return nil, false
	}

	return info, true
}

// parsePathExt splits a Windows PATHEXT value into a lowercased extension list
// (e.g., [".com", ".exe", ".bat", ".cmd"]). Returns nil when isWindows is
// false. Falls back to a conservative default when isWindows is true but
// pathExt is unset. isWindows is taken as a parameter so both branches are
// exercised by tests on any host.
func parsePathExt(pathExt string, isWindows bool) []string {
	if !isWindows {
		return nil
	}
	if pathExt == "" {
		pathExt = ".COM;.EXE;.BAT;.CMD"
	}
	parts := strings.Split(pathExt, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

// runPlugin executes the plugin with the given args, wiring stdio to the
// provided streams, and returns the child's exit code. ctx is propagated to
// exec.CommandContext so callers can cancel the child (e.g., via a future
// timeout); pass context.Background() when no cancellation is needed. A
// failure to start the child (as opposed to the child exiting non-zero)
// writes a diagnostic to stderr and returns exitPluginStartFailure, matching
// the shell convention for "command not found / not executable".
func runPlugin(ctx context.Context, p Plugin, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// #nosec G204 G702 -- plugin dispatch is the entire purpose of this function
	// (#95 Phase 1, git/kubectl model). p.Path was resolved by discoverPlugins from
	// a PATH directory and always contains a separator, so exec.LookPath/PATH-routed
	// resolution does not occur; args are user input forwarded as a string slice
	// (no shell). Trust model is "user controls PATH", documented in the PR.
	// G702 (taint analysis) is listed alongside G204 because gosec reports the
	// subprocess-with-variable-input pattern under both rule IDs; suppressing only
	// G204 leaves the taint rule firing on the same line.
	cmd := exec.CommandContext(ctx, p.Path, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		return ee.ExitCode()
	}
	_, _ = fmt.Fprintf(stderr, "glx: failed to run plugin %q: %v\n", p.Name, err)

	return exitPluginStartFailure
}

// listPlugins writes a human-readable listing of discovered plugins to out.
// Plugins whose names match a built-in command are annotated as shadowed —
// they cannot be invoked via `glx <name>` because built-in commands take
// precedence. When no plugins are discovered, an explanatory notice is written
// instead so the user knows how to install one — unless quiet is true, in
// which case the empty-case notice is suppressed to honor the established
// `--quiet`/`-q` contract for informational output. The listing itself (when
// non-empty) is the user-requested data, analogous to `glx add`'s
// `IOStreams.MachineOut`, and is always printed regardless of quiet.
func listPlugins(plugins []Plugin, known map[string]bool, quiet bool, out io.Writer) {
	if len(plugins) == 0 {
		if !quiet {
			_, _ = fmt.Fprintln(out, "No glx plugins found on PATH. Plugins are executables named glx-<name> placed on your PATH.")
		}

		return
	}
	w := len("NAME")
	for _, p := range plugins {
		if n := len(p.Name); n > w {
			w = n
		}
	}
	_, _ = fmt.Fprintf(out, "%-*s  %s\n", w, "NAME", "PATH")
	for _, p := range plugins {
		line := fmt.Sprintf("%-*s  %s", w, p.Name, p.Path)
		if known[p.Name] {
			line += "  (shadowed by built-in command)"
		}
		_, _ = fmt.Fprintln(out, line)
	}
}

// ensureBuiltinSubcommands materializes cobra's lazily-added "help" and
// "completion" subcommands on root so that subsequent enumeration via
// knownCommandNames sees them. Calling this is idempotent in cobra v1.10.
// It's separated from knownCommandNames so the latter can remain a pure read.
func ensureBuiltinSubcommands(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
}

// knownCommandNames returns the set of built-in command names and aliases of
// the root command tree. The caller is expected to have invoked
// ensureBuiltinSubcommands(root) earlier so that cobra's auto-added "help" and
// "completion" commands appear in root.Commands(). The "help"/"completion"
// fallback entries are belt-and-suspenders for cobra versions where the lazy
// init ordering differs.
func knownCommandNames(root *cobra.Command) map[string]bool {
	out := map[string]bool{}
	for _, c := range root.Commands() {
		out[c.Name()] = true
		for _, a := range c.Aliases {
			out[a] = true
		}
	}
	out["help"] = true
	out["completion"] = true

	return out
}

// pluginInvocation contains only the options GLX owns. args starts at the
// first positional argument; the plugin's arguments are never parsed here.
type pluginInvocation struct {
	args        []string
	list        bool
	quiet       bool
	help        bool
	version     bool
	pluginsFlag bool
}

// parsePluginInvocation uses pflag's normal boolean and shorthand syntax
// without touching Cobra's shared flag values. If GLX gains another global
// option, register it here too before allowing it on the plugin path.
func parsePluginInvocation(args []string) (pluginInvocation, bool) {
	var invocation pluginInvocation
	flags := pflag.NewFlagSet("glx plugin prefix", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.SetInterspersed(false)
	flags.BoolVarP(&invocation.quiet, rootFlagQuiet, quietShorthand, false, "")
	flags.BoolVar(&invocation.list, rootFlagPlugins, false, "")
	flags.BoolVarP(&invocation.help, "help", "h", false, "")
	flags.BoolVarP(&invocation.version, "version", "v", false, "")
	if err := flags.Parse(args); err != nil || flags.ArgsLenAtDash() >= 0 {
		return pluginInvocation{}, false
	}
	invocation.args = flags.Args()
	invocation.pluginsFlag = flags.Changed(rootFlagPlugins)

	return invocation, true
}

// pluginDispatchTarget takes the positional arguments after the GLX prefix and
// decides whether `glx <args>` should be redirected to a
// discovered plugin instead of letting cobra handle it. It returns the plugin
// to invoke and the args to forward, or ok=false if run() should fall
// through to cobra (which will handle the built-in, print "unknown command",
// or render help). Cobra-internal completion commands (the `__complete*`
// family — `__complete` and `__completeNoDesc` in cobra v1.10) and built-in
// commands always win — built-ins take precedence so a `glx-validate` plugin
// can never preempt the bundled `validate` command.
//
// Only the `__complete` prefix is reserved (matching cobra's actual
// `ShellCompRequestCmd` / `ShellCompNoDescRequestCmd`), not the broader `__`
// space, so a hypothetical `glx-__foo` plugin remains dispatchable. If cobra
// ever introduces another reserved completion command outside this prefix,
// add it to knownCommandNames so it falls through to cobra normally.
//
// On Windows (isWindows=true) the lookup name is lowercased before checking
// against built-ins and discovered plugins. discoverPlugins also stores
// plugin names lowercased on Windows, and Windows filesystems are
// case-insensitive, so `glx Hello` correctly finds `glx-hello.exe` and a
// `glx-validate` plugin cannot shadow the bundled `validate` command just
// because the user typed `glx VALIDATE`.
func pluginDispatchTarget(root *cobra.Command, args []string, pathEnv, pathExt string, isWindows bool) (Plugin, []string, bool) {
	if len(args) == 0 {
		return Plugin{}, nil, false
	}
	name := args[0]
	if strings.HasPrefix(name, "__complete") {
		return Plugin{}, nil, false
	}
	lookup := name
	if isWindows {
		lookup = strings.ToLower(name)
	}
	if knownCommandNames(root)[lookup] {
		return Plugin{}, nil, false
	}
	p, found := findPlugin(lookup, pathEnv, pathExt)
	if !found {
		return Plugin{}, nil, false
	}

	return p, args[1:], true
}
