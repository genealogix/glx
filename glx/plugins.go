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

	// Path is the path to the executable that will be exec'd when `glx <Name>` is
	// invoked. It is filepath.Join(<PATH dir>, <filename>), so it is absolute when
	// the PATH directory was absolute.
	Path string
}

// discoverPlugins scans the directories in pathEnv (the PATH environment variable)
// for executables named glx-<name> and returns them sorted by Name. When the same
// plugin name appears in more than one PATH directory the first occurrence wins,
// matching git's plugin resolution.
//
// pathExt is consulted on Windows to determine which file extensions count as
// executable (PATHEXT, default ".COM;.EXE;.BAT;.CMD"); it is ignored on other
// platforms, where the executable bit is checked instead. Directories are
// skipped. Unreadable or non-existent PATH directories are skipped silently,
// mirroring shell behavior.
func discoverPlugins(pathEnv, pathExt string) []Plugin {
	exts := parsePathExt(pathExt)
	seen := make(map[string]struct{})
	var plugins []Plugin
	for _, dir := range filepath.SplitList(pathEnv) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name, ok := pluginNameFromEntry(e, exts)
			if !ok {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			plugins = append(plugins, Plugin{Name: name, Path: filepath.Join(dir, e.Name())})
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
// glx- prefix and is executable on the current platform. On Windows that means
// the extension is in PATHEXT; on other platforms the file must have at least
// one executable permission bit set.
func pluginNameFromEntry(e os.DirEntry, exts []string) (string, bool) {
	if e.IsDir() {
		return "", false
	}
	fn := e.Name()
	matchName := fn
	if runtime.GOOS == osWindows {
		matchName = strings.ToLower(fn)
	}
	if !strings.HasPrefix(matchName, pluginPrefix) {
		return "", false
	}
	if runtime.GOOS == osWindows {
		for _, ext := range exts {
			if strings.HasSuffix(matchName, ext) {
				name := strings.TrimSuffix(strings.TrimPrefix(matchName, pluginPrefix), ext)
				if name == "" {
					return "", false
				}

				return name, true
			}
		}

		return "", false
	}
	info, err := e.Info()
	if err != nil || info.Mode()&0o111 == 0 {
		return "", false
	}
	name := strings.TrimPrefix(fn, pluginPrefix)
	if name == "" {
		return "", false
	}

	return name, true
}

// parsePathExt splits a Windows PATHEXT value into a lowercased extension list
// (e.g., [".com", ".exe", ".bat", ".cmd"]). Returns nil on non-Windows. Falls
// back to a conservative default when PATHEXT is unset on Windows.
func parsePathExt(pathExt string) []string {
	if runtime.GOOS != osWindows {
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
// instead so the user knows how to install one.
func listPlugins(plugins []Plugin, known map[string]bool, out io.Writer) {
	if len(plugins) == 0 {
		_, _ = fmt.Fprintln(out, "No glx plugins found on PATH. Plugins are executables named glx-<name> placed on your PATH.")

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

// knownCommandNames returns the set of built-in command names and aliases of the
// root command tree, including cobra's auto-added "help" and "completion"
// commands. Used to decide whether `glx <name>` should dispatch to a plugin
// (no when name is a built-in).
func knownCommandNames(root *cobra.Command) map[string]bool {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	out := map[string]bool{}
	for _, c := range root.Commands() {
		out[c.Name()] = true
		for _, a := range c.Aliases {
			out[a] = true
		}
	}
	// Belt-and-suspenders: cobra adds these lazily depending on call order.
	out["help"] = true
	out["completion"] = true

	return out
}

// firstSubcommandToken returns the first non-flag token in args (the candidate
// subcommand name) and the args that follow it. Tokens starting with "-" are
// skipped on the way to the first positional argument; this matches how git
// resolves its subcommand when global flags appear before it. Args before the
// resolved token are NOT forwarded to a plugin in Phase 1 — a deliberate
// simplification that mirrors git's behavior with global flags.
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
		if a == "--plugins" {
			return true
		}
		if rest, ok := strings.CutPrefix(a, "--plugins="); ok {
			if b, err := strconv.ParseBool(rest); err == nil && b {
				return true
			}
		}
	}

	return false
}
