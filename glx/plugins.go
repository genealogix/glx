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
	pluginsFlag = "--plugins"
	quietFlag   = "--quiet"
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
			name, extIdx, ok := pluginNameFromEntry(e, exts, isWindows)
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

// findPlugin returns the discovered plugin for the given subcommand name, if any.
func findPlugin(name, pathEnv, pathExt string) (Plugin, bool) {
	for _, p := range discoverPlugins(pathEnv, pathExt) {
		if p.Name == name {
			return p, true
		}
	}

	return Plugin{}, false
}

// pluginNameFromEntry extracts a plugin's logical name from a directory entry.
// Returns ok=false unless the entry is a regular file (not a directory) with the
// glx- prefix and is executable on the target platform. When isWindows is true,
// the entry's extension must be in exts; the returned extIdx is the matched
// extension's index in exts (lower = higher PATHEXT priority) and is used by
// discoverPlugins to pick the right file when several extensions co-exist.
// Otherwise the file must have at least one executable permission bit set;
// extIdx is always 0.
//
// isWindows is taken as a parameter (rather than read from runtime.GOOS inside
// the function) so both platform branches are exercised by unit tests on any
// host.
func pluginNameFromEntry(e os.DirEntry, exts []string, isWindows bool) (name string, extIdx int, ok bool) {
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
	info, err := e.Info()
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		// Reject directories (already filtered above via e.IsDir(), but defense
		// in depth), symlinks (e.Info() reports the link's own mode, which is
		// not regular), and devices/sockets — none of these are safe to exec
		// as a plugin even if their mode bits happen to include execute. A
		// symlink that points to a legitimate executable falls under the same
		// rule; supporting that is a deliberate Phase-2 follow-up if needed.
		return "", 0, false
	}
	stem := strings.TrimPrefix(fn, pluginPrefix)
	if stem == "" {
		return "", 0, false
	}

	return stem, 0, true
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
	// #nosec G204 -- plugin dispatch is the entire purpose of this function (#95
	// Phase 1, git/kubectl model). p.Path was resolved by discoverPlugins from a
	// PATH directory and always contains a separator, so exec.LookPath/PATH-routed
	// resolution does not occur; args are user input forwarded as a string slice
	// (no shell). Trust model is "user controls PATH", documented in the PR.
	cmd := exec.CommandContext(ctx, p.Path, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
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

// firstSubcommandToken returns the first non-flag token in args (the candidate
// subcommand name) and the args that follow it. Tokens starting with "-" are
// skipped on the way to the first positional argument; this matches how git
// resolves its subcommand when global flags appear before it.
//
// IMPORTANT (tripwire for future contributors): this helper assumes every
// global/root flag is a boolean (no value follows it). If a value-taking
// persistent root flag is ever added (e.g., `--config <path>`), this scan will
// misread the value as the subcommand. Adding such a flag MUST be paired with
// teaching this function to consult `rootCmd.PersistentFlags()` so it can skip
// past the value of value-taking flags. Today's root flags (`-q`/`--quiet`,
// `--plugins`) are both booleans, so the simple scan is correct.
//
// Args before the resolved token are NOT forwarded to a plugin in Phase 1 — a
// deliberate simplification that mirrors git's behavior with global flags.
func firstSubcommandToken(args []string) (name string, rest []string, ok bool) {
	for i, a := range args {
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}

		return a, args[i+1:], true
	}

	return "", nil, false
}

// pluginsFlagRequested reports whether the user invoked `glx --plugins` with no
// subcommand attached. Both the bare flag form (`--plugins`) and the
// value-attached forms cobra accepts for bool flags (`--plugins=true`,
// `--plugins=1`, etc.) are recognized; falsey values fall through so they
// reach cobra's normal flag parsing path. The check runs before cobra parses
// args because rootCmd has no Run handler — making it runnable to read the
// flag via RunE would suppress cobra's standard "unknown command" error for
// typos. When a subcommand IS present, --plugins is ignored and the
// subcommand wins.
func pluginsFlagRequested(args []string) bool {
	if _, _, hasSub := firstSubcommandToken(args); hasSub {
		return false
	}
	for _, a := range args {
		if a == pluginsFlag {
			return true
		}
		if rest, ok := strings.CutPrefix(a, pluginsFlag+"="); ok {
			if b, err := strconv.ParseBool(rest); err == nil && b {
				return true
			}
		}
	}

	return false
}

// quietFlagRequested reports whether `-q`, `--quiet`, or a truthy
// `--quiet=true|1|t|...` was passed in args. Used by Execute() to honor the
// established `--quiet` contract for the `glx --plugins` listing path, which
// runs before cobra parses flags into the package-level quietOutput variable.
func quietFlagRequested(args []string) bool {
	for _, a := range args {
		if a == "-q" || a == quietFlag {
			return true
		}
		if rest, ok := strings.CutPrefix(a, quietFlag+"="); ok {
			if b, err := strconv.ParseBool(rest); err == nil && b {
				return true
			}
		}
	}

	return false
}

// pluginDispatchTarget decides whether `glx <args>` should be redirected to a
// discovered plugin instead of letting cobra handle it. It returns the plugin
// to invoke and the args to forward, or ok=false if Execute() should fall
// through to cobra (which will handle the built-in, print "unknown command",
// or render help). Cobra-internal completion commands (the `__complete*`
// family) and built-in commands always win — built-ins take precedence so a
// `glx-validate` plugin can never preempt the bundled `validate` command.
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
	if strings.HasPrefix(name, "__") {
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
