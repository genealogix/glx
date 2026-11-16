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

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func connectTestServer(t *testing.T, server *mcpsdk.Server) *mcpsdk.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	ss, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ss.Close()) })
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "1"}, nil)
	cs, err := client.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cs.Close()) })

	return cs
}

func TestServerCatalogAndStructuredTools(t *testing.T) {
	var calls atomic.Int32
	server, err := NewServer(BackendFunc(func(_ context.Context, name string, args json.RawMessage) (any, error) {
		calls.Add(1)

		return map[string]any{"operation": name, "arguments": args}, nil
	}), "test")
	require.NoError(t, err)
	client := connectTestServer(t, server)
	catalog, err := client.ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, catalog.Tools, 14)
	for _, tool := range catalog.Tools {
		require.True(t, tool.Annotations.ReadOnlyHint, tool.Name)
		require.NotNil(t, tool.InputSchema)
	}
	for name, args := range map[string]any{
		"query":  QueryArgs{EntityType: "persons", Name: "Mary"},
		"vitals": PersonArgs{Person: "person-mary"}, "summary": PersonArgs{Person: "person-mary"},
		"timeline": TimelineArgs{Person: "person-mary"}, "ancestors": TreeArgs{Person: "person-mary"}, "descendants": TreeArgs{Person: "person-mary"},
		"path": PathArgs{From: "a", To: "b"}, "cite": CiteArgs{Citation: "citation-1"},
		"analyze": AnalyzeArgs{Check: "gaps"}, "stats": struct{}{}, "places": struct{}{},
		"cluster": ClusterArgs{Person: "person-mary"}, "validate": ValidateArgs{Report: true}, "list_entity_types": struct{}{},
	} {
		result, callErr := client.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: name, Arguments: args})
		require.NoError(t, callErr, name)
		require.False(t, result.IsError, name)
		require.NotNil(t, result.StructuredContent, name)
		require.Len(t, result.Content, 1, name)
		text, ok := result.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		require.True(t, json.Valid([]byte(text.Text)))
	}
	require.EqualValues(t, 14, calls.Load())
	for _, args := range []any{map[string]any{}, map[string]any{"person": 42}, map[string]any{"person": "a", "archive": "/etc/passwd"}} {
		result, callErr := client.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "vitals", Arguments: args})
		require.True(t, callErr != nil || result.IsError)
	}
	require.EqualValues(t, 14, calls.Load(), "invalid arguments must not reach the backend")
}

var errTestPersonNotFound = errors.New("person not found")

func TestServerErrorsResourcesPromptsAndExtension(t *testing.T) {
	server, err := NewServer(BackendFunc(func(_ context.Context, name string, args json.RawMessage) (any, error) {
		if name == "vitals" {
			return nil, errTestPersonNotFound
		}

		return map[string]any{"operation": name}, nil
	}), "")
	require.NoError(t, err)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "custom_search", Description: "Downstream extension"}, func(_ context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"result": "custom"}, nil
	})
	client := connectTestServer(t, server)
	result, err := client.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "vitals", Arguments: PersonArgs{Person: "missing"}})
	require.NoError(t, err)
	require.True(t, result.IsError)
	custom, err := client.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "custom_search", Arguments: struct{}{}})
	require.NoError(t, err)
	require.False(t, custom.IsError)
	for _, uri := range []string{"glx://archive/stats", "glx://archive/vocabularies"} {
		resource, readErr := client.ReadResource(t.Context(), &mcpsdk.ReadResourceParams{URI: uri})
		require.NoError(t, readErr)
		require.Len(t, resource.Contents, 1)
		require.Equal(t, "application/json", resource.Contents[0].MIMEType)
		require.True(t, json.Valid([]byte(resource.Contents[0].Text)))
	}
	prompt, err := client.GetPrompt(t.Context(), &mcpsdk.GetPromptParams{Name: "genealogy-research-assistant"})
	require.NoError(t, err)
	require.Len(t, prompt.Messages, 1)
}

func TestSerialBackendAndCancellation(t *testing.T) {
	var active atomic.Int32
	b := &serialBackend{Backend: BackendFunc(func(ctx context.Context, name string, args json.RawMessage) (any, error) {
		require.EqualValues(t, 1, active.Add(1))
		defer active.Add(-1)

		return map[string]any{}, nil
	}), gate: make(chan struct{}, 1)}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, err := b.CallTool(t.Context(), "stats", nil); require.NoError(t, err) })
	}
	wg.Wait()
	b.gate <- struct{}{}
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	_, err := b.CallTool(ctx, "stats", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	<-b.gate
}

func TestNilBackend(t *testing.T) {
	_, err := NewServer(nil, "")
	require.Error(t, err)
}

func TestBackendPanicIsToolErrorAndSessionSurvives(t *testing.T) {
	server, err := NewServer(BackendFunc(func(_ context.Context, name string, _ json.RawMessage) (any, error) {
		if name == "vitals" {
			panic("untrusted panic detail")
		}

		return map[string]any{"ok": true}, nil
	}), "")
	require.NoError(t, err)
	client := connectTestServer(t, server)
	result, err := client.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "vitals", Arguments: PersonArgs{Person: "p"}})
	require.NoError(t, err)
	require.True(t, result.IsError)
	data, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(data), "untrusted panic detail")
	result, err = client.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "stats", Arguments: struct{}{}})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestSerialBackendDiscardsResultAfterCancellation(t *testing.T) {
	for _, backendErr := range []error{nil, errTestPersonNotFound} {
		ctx, cancel := context.WithCancel(t.Context())
		b := &serialBackend{Backend: BackendFunc(func(context.Context, string, json.RawMessage) (any, error) {
			// Model an operation that completes its current unit of work after
			// cancellation instead of checking context in every domain loop.
			cancel()

			return map[string]bool{"finished": true}, backendErr
		}), gate: make(chan struct{}, 1)}
		result, err := b.CallTool(ctx, "analyze", nil)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, result, "canceled work must not produce a successful/stale result")
	}
}

func TestSerialBackendCancellationKeepsGateUntilWorkEnds(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseWork := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseWork)
	var calls atomic.Int32
	b := &serialBackend{Backend: BackendFunc(func(context.Context, string, json.RawMessage) (any, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}

		return map[string]bool{"finished": true}, nil
	}), gate: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		_, err := b.CallTool(ctx, "analyze", nil)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("backend did not start")
	}
	cancel()
	waiting, stopWaiting := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer stopWaiting()
	_, err := b.CallTool(waiting, "stats", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.EqualValues(t, 1, calls.Load(), "cancellation must not let another request race the unfinished backend")
	select {
	case <-done:
		t.Fatal("backend work was abandoned while still running")
	default:
	}
	releaseWork()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("backend did not finish after release")
	}
	result, err := b.CallTool(t.Context(), "stats", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.EqualValues(t, 2, calls.Load(), "gate must be reusable once the canceled operation actually finishes")
}

type cancelDuringJSON struct{ cancel context.CancelFunc }

func (value cancelDuringJSON) MarshalJSON() ([]byte, error) {
	value.cancel()

	return []byte(`{"finished":true}`), nil
}

func TestMarshalResultCancellationDuringEncoding(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	data, err := marshalResult(ctx, cancelDuringJSON{cancel: cancel})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, data)
}
