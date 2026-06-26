package eino

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestStreamClient(server *httptest.Server) *openAICompatibleClient {
	return &openAICompatibleClient{
		config: LLMConfig{
			Provider: "deepseek",
			BaseURL:  server.URL,
			Model:    "deepseek-v4-flash",
			APIKey:   "test-key",
			Timeout:  5 * time.Second,
		},
		httpClient: server.Client(),
	}
}

func sseFrame(payload string) string {
	return fmt.Sprintf("data: %s\n\n", payload)
}

func TestChatCompletionStream_NormalContentDelta(t *testing.T) {
	frames := []string{
		`{"choices":[{"index":0,"delta":{"content":"你"},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"content":"好"},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"content":"，"},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"content":"世"},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"content":"界"},"finish_reason":"stop"}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, frame := range frames {
			fmt.Fprint(w, sseFrame(frame))
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := newTestStreamClient(server)
	var deltas []string
	var mu sync.Mutex
	result, err := client.chatCompletionStream(context.Background(), chatCompletionRequest{Model: "deepseek-v4-flash"}, func(delta string) {
		mu.Lock()
		deltas = append(deltas, delta)
		mu.Unlock()
	}, nil)
	if err != nil {
		t.Fatalf("chatCompletionStream returned error: %v", err)
	}
	if got := result.Content; got != "你好，世界" {
		t.Fatalf("accumulated content mismatch: got %q", got)
	}
	if len(deltas) != 5 {
		t.Fatalf("expected 5 onDelta calls, got %d (%v)", len(deltas), deltas)
	}
	if result.FinishReason != "stop" {
		t.Fatalf("expected finish_reason=stop, got %q", result.FinishReason)
	}
}

func TestChatCompletionStream_ToolCallArgumentsAccumulate(t *testing.T) {
	frames := []string{
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"submit_travel_plan","arguments":"{\"title\":"}}]},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Hangzhou\","}}]},"finish_reason":null}]}`,
		`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"days\":3}"}}]},"finish_reason":"tool_calls"}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, frame := range frames {
			fmt.Fprint(w, sseFrame(frame))
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := newTestStreamClient(server)
	var deltaCalls int
	result, err := client.chatCompletionStream(context.Background(), chatCompletionRequest{Model: "deepseek-v4-flash"}, func(string) {
		deltaCalls++
	}, nil)
	if err != nil {
		t.Fatalf("chatCompletionStream returned error: %v", err)
	}
	if got, want := result.ToolCallArguments[0], `{"title":"Hangzhou","days":3}`; got != want {
		t.Fatalf("tool call arguments mismatch:\n got=%q\nwant=%q", got, want)
	}
	if got := result.ToolCallNames[0]; got != "submit_travel_plan" {
		t.Fatalf("tool call name mismatch: got %q", got)
	}
	if deltaCalls != 0 {
		t.Fatalf("onDelta should not be called for tool call fragments, got %d calls", deltaCalls)
	}
}

func TestChatCompletionStream_DoneTerminator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, sseFrame(`{"choices":[{"delta":{"content":"hi"}}]}`))
		fmt.Fprint(w, "data: [DONE]\n\n")
		// Anything after [DONE] must be ignored — also tests early-exit.
		fmt.Fprint(w, sseFrame(`{"choices":[{"delta":{"content":"IGNORED"}}]}`))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := newTestStreamClient(server)
	result, err := client.chatCompletionStream(context.Background(), chatCompletionRequest{Model: "deepseek-v4-flash"}, nil, nil)
	if err != nil {
		t.Fatalf("chatCompletionStream returned error: %v", err)
	}
	if result.Content != "hi" {
		t.Fatalf("expected content=hi, got %q", result.Content)
	}
}

// flusher that allows writing partial frames split across packets.
type slowFlushWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (s *slowFlushWriter) writePartial(payload string) {
	fmt.Fprint(s.w, payload)
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func TestChatCompletionStream_PartialFramesAcrossPackets(t *testing.T) {
	// Split a single SSE event across two flushes, simulating a TCP packet boundary
	// in the middle of the JSON payload.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		writer := &slowFlushWriter{w: w, flusher: flusher}
		writer.writePartial("data: {\"choices\":[{\"delta\":{\"con")
		time.Sleep(20 * time.Millisecond)
		writer.writePartial("tent\":\"ok\"}}]}\n\n")
		writer.writePartial("data: [DONE]\n\n")
	}))
	defer server.Close()

	client := newTestStreamClient(server)
	result, err := client.chatCompletionStream(context.Background(), chatCompletionRequest{Model: "deepseek-v4-flash"}, nil, nil)
	if err != nil {
		t.Fatalf("chatCompletionStream returned error: %v", err)
	}
	if result.Content != "ok" {
		t.Fatalf("expected content=ok, got %q", result.Content)
	}
}

func TestChatCompletionStream_HTTP5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "upstream blew up")
	}))
	defer server.Close()

	client := newTestStreamClient(server)
	_, err := client.chatCompletionStream(context.Background(), chatCompletionRequest{Model: "deepseek-v4-flash"}, nil, nil)
	if err == nil {
		t.Fatal("expected error for 5xx response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected error to include status 500, got %v", err)
	}
}

func TestChatCompletionStream_ContextCancel(t *testing.T) {
	// Server hangs; client cancels ctx; chatCompletionStream must return
	// promptly with ctx.Err().
	releaseServer := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, sseFrame(`{"choices":[{"delta":{"content":"slow"}}]}`))
		if flusher != nil {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
		case <-releaseServer:
		}
	}))
	defer func() {
		close(releaseServer)
		server.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	client := newTestStreamClient(server)
	errCh := make(chan error, 1)
	go func() {
		_, err := client.chatCompletionStream(ctx, chatCompletionRequest{Model: "deepseek-v4-flash"}, nil, nil)
		errCh <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected ctx.Err() after cancel, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("chatCompletionStream did not return within 2s after ctx cancel")
	}
}

func TestChatCompletionStream_EmptyOrFinishOnlyFrames(t *testing.T) {
	frames := []string{
		`{"choices":[{"delta":{}}]}`,
		`{"choices":[{"delta":{"content":""}}]}`,
		`{"choices":[{"delta":{"content":"actual"}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, frame := range frames {
			fmt.Fprint(w, sseFrame(frame))
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := newTestStreamClient(server)
	var deltaCalls int
	result, err := client.chatCompletionStream(context.Background(), chatCompletionRequest{Model: "deepseek-v4-flash"}, func(string) {
		deltaCalls++
	}, nil)
	if err != nil {
		t.Fatalf("chatCompletionStream returned error: %v", err)
	}
	if result.Content != "actual" {
		t.Fatalf("expected content=actual, got %q", result.Content)
	}
	if deltaCalls != 1 {
		t.Fatalf("expected 1 onDelta call (skip empty content), got %d", deltaCalls)
	}
	if result.FinishReason != "stop" {
		t.Fatalf("expected finish_reason=stop, got %q", result.FinishReason)
	}
}
