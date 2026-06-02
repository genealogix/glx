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

// siteHandler serves the generated static site from dir.
func siteHandler(dir string) http.Handler {
	return http.FileServer(http.Dir(dir))
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

	server := &http.Server{
		Handler:           siteHandler(dir),
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	streams.Printf("\nServing %s\n  → http://%s/\nPress Ctrl+C to stop.\n", dir, listener.Addr().String())

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
