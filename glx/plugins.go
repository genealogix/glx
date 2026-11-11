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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
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

// pluginsFlag and quietFlag are the long-form CLI flag tokens recognized by the
// pre-cobra interception path in Execute(). Hoisted to named constants to
// satisfy the goconst linter (each token recurs across this file and its tests)
// and to keep the flag-string spellings single-sourced. The `=`-attached forms
// used for value-bearing parses are derived as `pluginsFlag + "="` /
// `quietFlag + "="` at the call sites.
const (
	pluginsFlag    = "--plugins"
	quietFlag      = "--quiet"
	quietShortFlag = "-q"
)

// rootFlagPlugins and rootFlagQuiet are the dash-less flag names rootBoolFlag
// reports, so callers can tell which root flag a token set.
const (
	rootFlagPlugins = "plugins"
	rootFlagQuiet   = "quiet"
)

// quietValuePrefixes are the value-attached spellings pflag accepts for the
// boolean quiet flag. pflag's short-argument parser checks for an `=` before it
// falls back to a bool's implicit "true", so `-q=true` and `-q=false` are both
// valid invocations alongside the long `--quiet=...` forms. Package-level so the
// list is not rebuilt for every argument scanned.
var quietValuePrefixes = [...]string{quietFlag + "=", quietShortFlag + "="}

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

// endOfFlags is the conventional end-of-options marker. pflag stops parsing
// flags at it and cobra's subcommand lookup (stripFlags) stops there too, so
// every token after it is positional and none of them names a subcommand.
const endOfFlags = "--"

// firstSubcommandToken returns the first non-flag token in args (the candidate
// subcommand name) and the args that follow it. Tokens starting with "-" are
// skipped on the way to the first positional argument; this matches how git
// resolves its subcommand when global flags appear before it.
//
// The scan stops at the `--` end-of-flags marker and reports no subcommand,
// matching cobra: `glx -- foo` does not run a `foo` subcommand, so it must not
// dispatch a `glx-foo` plugin either, and `glx -- --plugins` must not be read
// as the --plugins flag.
//
// IMPORTANT (tripwire for future contributors): this helper assumes every
// global/root flag is a boolean (no value follows it). If a value-taking
// persistent root flag is ever added (e.g., `--config <path>`), this scan will
// misread the value as the subcommand. Adding such a flag MUST be paired with
// teaching this function to consult `rootCmd.PersistentFlags()` so it can skip
// past the value of value-taking flags. Today's root flags (`-q`/`--quiet`,
// `--plugins`) are both booleans, so the simple scan is correct. Note that a
// boolean flag never consumes a following token: pflag gives bool flags an
// implicit value, so `glx --quiet false` sets quiet and treats `false` as the
// subcommand (cobra reports `unknown command "false"`). This scan agrees.
//
// Args before the resolved token are NOT forwarded to a plugin in Phase 1 — a
// deliberate simplification that mirrors git's behavior with global flags.
func firstSubcommandToken(args []string) (name string, rest []string, ok bool) {
	for i, a := range args {
		if a == endOfFlags {
			return "", nil, false
		}
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}

		return a, args[i+1:], true
	}

	return "", nil, false
}

// rootBoolFlag recognizes one spelling of the root command's boolean flags as
// pflag would parse it: the bare forms (`--plugins`, `--quiet`, `-q`) and the
// value-attached forms (`--plugins=<bool>`, `--quiet=<bool>`, `-q=<bool>`).
// It returns the flag's long name without dashes and the value pflag would
// assign. ok is false for anything else — a positional token, `--`, an
// unrecognized flag, or a value-attached form whose value pflag would reject
// (which cobra reports as an invalid argument).
func rootBoolFlag(a string) (name string, value, ok bool) {
	switch a {
	case pluginsFlag:
		return rootFlagPlugins, true, true
	case quietFlag, quietShortFlag:
		return rootFlagQuiet, true, true
	}
	if raw, found := strings.CutPrefix(a, pluginsFlag+"="); found {
		b, err := strconv.ParseBool(raw)

		return rootFlagPlugins, b, err == nil
	}
	for _, prefix := range quietValuePrefixes {
		if raw, found := strings.CutPrefix(a, prefix); found {
			b, err := strconv.ParseBool(raw)

			return rootFlagQuiet, b, err == nil
		}
	}

	return "", false, false
}

// pluginsFlagRequested reports whether `glx <args>` is a request to list
// plugins: --plugins resolves to true and nothing else on the command line
// would make cobra do something different. Every token must be a recognized
// spelling of a root boolean flag (see rootBoolFlag); a subcommand, the `--`
// marker, an unknown flag (`glx --plugins --bogus`), or an unparseable value
// makes this return false so cobra reports the problem with its usual error
// instead of the listing silently swallowing it. When a flag is repeated the
// last occurrence wins, as it does in pflag (`--plugins --plugins=false` does
// not list).
//
// The check runs before cobra parses args because rootCmd has no Run handler —
// making it runnable to read the flag via RunE would suppress cobra's standard
// "unknown command" error for typos.
func pluginsFlagRequested(args []string) bool {
	requested := false
	for _, a := range args {
		name, value, ok := rootBoolFlag(a)
		if !ok {
			return false
		}
		if name == rootFlagPlugins {
			requested = value
		}
	}

	return requested
}

// quietFlagRequested reports the value pflag would give --quiet for args: each
// recognized spelling (`-q`, `--quiet`, `--quiet=<bool>`, `-q=<bool>`) sets it,
// and the last one wins, so `-q --quiet=false` is not quiet. Both the long and
// shorthand value-attached spellings are recognized because pflag accepts
// both. Scanning stops at the `--` end-of-flags marker, after which pflag reads
// nothing as a flag.
//
// Used by Execute() to honor the established `--quiet` contract for the
// `glx --plugins` listing path, which runs before cobra parses flags into the
// package-level quietOutput variable.
func quietFlagRequested(args []string) bool {
	quiet := false
	for _, a := range args {
		if a == endOfFlags {
			break
		}
		if name, value, ok := rootBoolFlag(a); ok && name == rootFlagQuiet {
			quiet = value
		}
	}

	return quiet
}

// pluginDispatchTarget decides whether `glx <args>` should be redirected to a
// discovered plugin instead of letting cobra handle it. It returns the plugin
// to invoke and the args to forward, or ok=false if Execute() should fall
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
	name, rest, ok := firstSubcommandToken(args)
	if !ok {
		return Plugin{}, nil, false
	}
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

	return p, rest, true
}
