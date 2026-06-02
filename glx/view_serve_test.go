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
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// errTestServe is a static sentinel used to exercise ignoreServerClosed.
var errTestServe = errors.New("boom")

// testListen opens a loopback listener on an ephemeral port using a
// context-aware ListenConfig (satisfying the noctx linter).
func testListen(t *testing.T) net.Listener {
	t.Helper()
	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	return listener
}

func TestNewSiteServer_ServesAndShutsDown(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), []byte("<h1>served</h1>"))

	listener := testListen(t)

	server := newSiteServer(dir)
	errCh := make(chan error, 1)
	go func() { errCh <- ignoreServerClosed(server.Serve(listener)) }()

	url := "http://" + listener.Addr().String() + "/index.html"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "served") {
		t.Errorf("unexpected body: %q", body)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Errorf("serve returned error on graceful shutdown: %v", err)
	}
}

func TestServeSite_BindError(t *testing.T) {
	// Occupy a port, then ask serveSite to bind the same one.
	listener := testListen(t)
	defer func() { _ = listener.Close() }()
	port := listener.Addr().(*net.TCPAddr).Port

	streams, _, _ := TestIOStreams()
	if err := serveSite(streams, t.TempDir(), port); err == nil {
		t.Error("expected an error binding an already-occupied port")
	}
}

func TestIgnoreServerClosed(t *testing.T) {
	if err := ignoreServerClosed(nil); err != nil {
		t.Errorf("nil -> %v, want nil", err)
	}
	if err := ignoreServerClosed(http.ErrServerClosed); err != nil {
		t.Errorf("ErrServerClosed -> %v, want nil", err)
	}
	if err := ignoreServerClosed(errTestServe); err == nil {
		t.Error("real error should propagate")
	}
}
