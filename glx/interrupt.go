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
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// exitInterrupted is the process exit code after SIGINT or SIGTERM: 128 plus
// SIGINT's number, the shell convention for "killed by Ctrl-C".
const exitInterrupted = 130

// interrupts holds the state the interrupt handler needs: the temporary files
// and directories glx has created and not yet renamed into place or removed,
// and a lock that keeps the handler out of critical sections.
var interrupts = struct {
	mu    sync.Mutex
	temps map[string]struct{}
	// critical is read-locked by holdInterrupts for the length of a section
	// that must not be cut short, and write-locked by the handler before it
	// cleans up, so an interrupt waits for every open section to finish.
	critical sync.RWMutex
	// stop ends the watch started by watchInterrupts. A command that handles
	// signals itself calls handOverInterrupts, which calls it.
	stop func()
}{temps: map[string]struct{}{}, stop: func() {}}

// trackTemp registers path, a temporary file or directory, for removal if glx
// is interrupted before the caller cleans it up. The returned func
// unregisters it; call it once the path has been renamed into place or
// removed.
func trackTemp(path string) (untrack func()) {
	interrupts.mu.Lock()
	interrupts.temps[path] = struct{}{}
	interrupts.mu.Unlock()

	return func() {
		interrupts.mu.Lock()
		delete(interrupts.temps, path)
		interrupts.mu.Unlock()
	}
}

// holdInterrupts defers any interrupt until the returned release is called.
// Wrap only short sections that would leave the archive harder to recover if
// cut off midway, such as moving entries between an archive and its backup.
func holdInterrupts() (release func()) {
	interrupts.critical.RLock()

	return interrupts.critical.RUnlock
}

// removeTrackedTemps deletes every path registered with trackTemp.
func removeTrackedTemps() {
	interrupts.mu.Lock()
	defer interrupts.mu.Unlock()
	for path := range interrupts.temps {
		_ = os.RemoveAll(path)
		delete(interrupts.temps, path)
	}
}

// watchInterrupts makes SIGINT and SIGTERM wait for any held section, remove
// the tracked temporary paths, report the interrupt on stderr and exit with
// exitInterrupted. Without it, Ctrl-C during an import or a split left
// .glx-tmp-* and .glx-stage-* entries inside the archive. The returned func
// stops the watch.
func watchInterrupts(stderr io.Writer) (stop func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	var once sync.Once
	stop = func() {
		once.Do(func() {
			signal.Stop(signals)
			close(done)
		})
	}
	interrupts.stop = stop

	go func() {
		select {
		case <-signals:
			// Never unlocked: the process exits below, and no section may
			// start in the meantime.
			interrupts.critical.Lock()
			removeTrackedTemps()
			_, _ = fmt.Fprintln(stderr, "glx: interrupted")
			os.Exit(exitInterrupted)
		case <-done:
		}
	}()

	return stop
}

// handOverInterrupts stops the default interrupt handling, for a command that
// installs its own (glx serve shuts down gracefully on SIGINT and SIGTERM).
func handOverInterrupts() {
	interrupts.stop()
}
