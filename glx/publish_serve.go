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
	"net"
	"net/http"
	"time"
)

// serverReadHeaderTimeout bounds how long the preview server waits for request
// headers, guarding against slow-client resource exhaustion.
const serverReadHeaderTimeout = 10 * time.Second

// siteHandler serves the generated static site from dir. It sets
// X-Content-Type-Options: nosniff as defense-in-depth so the preview server
// never MIME-sniffs a media file into an active document. (The primary defense
// is the publish-time allowlist, which also protects file:// and static hosts.)
func siteHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fileServer.ServeHTTP(w, r)
	})
}

// newSiteServer builds the HTTP server that serves the generated site from dir.
func newSiteServer(dir string) *http.Server {
	return &http.Server{
		Handler:           siteHandler(dir),
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}
}

// serveSite serves the generated site on 127.0.0.1:port until interrupted.
// A port of 0 selects an available ephemeral port. The chosen address is
// printed to streams.Out before the blocking serve loop begins.
func serveSite(streams *IOStreams, dir string, port int) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("failed to start server on port %d: %w", port, err)
	}

	server := newSiteServer(dir)
	streams.Printf("\nServing %s\n  → http://%s/\nPress Ctrl+C to stop.\n", dir, listener.Addr().String())

	return ignoreServerClosed(server.Serve(listener))
}

// ignoreServerClosed treats http.ErrServerClosed (the result of a graceful
// shutdown) as success and wraps any other error.
func ignoreServerClosed(err error) error {
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
