---
title: CLI Plugins
description: Install, run and write external commands for the GLX CLI.
---

# CLI Plugins

A GLX plugin is an executable named `glx-<name>` on your `PATH`. GLX runs it when you enter `glx <name>` and no built-in command or alias has that name.

## Install and run a plugin

Install the executable using the plugin author's instructions, or place it in a directory already on your `PATH`. On macOS and Linux, the file must be executable. On Windows, use a supported executable extension, such as `.exe`.

```bash
glx --plugins               # Show discovered plugins and their paths
glx -q --plugins            # Hide the notice when none are installed
glx hello --help            # Ask glx-hello for its help, if it provides it
glx hello "Jane Smith"      # Pass one argument containing a space
```

GLX handles its own flags before the plugin name. For example, `glx --help hello` shows GLX help instead of running the plugin. Use `glx hello -q` if you mean to pass `-q` to the plugin; a leading `-q` does not change the plugin's output. Everything after the plugin name, including `--`, is passed through unchanged. Use `--plugins` by itself or with `-q` to list plugins, not as a prefix to a command.

GLX uses the first matching executable on `PATH`. If a built-in takes the same name, the listing marks the plugin as shadowed. Choose a different name or run the executable directly.

> Only install plugins you trust. Plugins run with your account's permissions, in the current directory, and inherit your environment. GLX does not sandbox them.

## Write a plugin

For example, save this script as `glx-hello` on macOS or Linux, make it executable with `chmod +x glx-hello`, and put it on your `PATH`:

```sh
#!/bin/sh
name=${1:-world}
printf 'Hello, %s!\n' "$name"
```

Run `glx hello "Jane Smith"`. Use any language that can produce an executable for the target platform. The plugin owns its argument parsing and help. It uses the same standard input, output and error streams as GLX, and GLX returns its exit status.

GLX does not load or send an archive automatically. If a plugin needs one, it must accept a path or read standard input itself. There is no plugin manifest, validator/import/export protocol, hook system, SDK or registry in this release.

See the [CLI reference](../cli/index.md) for built-in commands.
