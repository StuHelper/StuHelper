package sse

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDisableWriteTimeoutAllowsDelayedSSEWrite(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := DisableWriteTimeout(w); err != nil {
			t.Errorf("DisableWriteTimeout() error = %v", err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if err := WriteComment(w, "connected"); err != nil {
			t.Errorf("WriteComment() error = %v", err)
			return
		}
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush() error = %v", err)
			return
		}
		time.Sleep(300 * time.Millisecond)
		if _, err := fmt.Fprint(w, "event: keepalive\ndata: ok\n\n"); err != nil {
			t.Errorf("delayed SSE write error = %v", err)
			return
		}
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("delayed Flush() error = %v", err)
		}
	})

	server := httptest.NewUnstartedServer(handler)
	server.Config.WriteTimeout = 100 * time.Millisecond
	server.Start()
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("GET SSE stream: %v", err)
	}
	defer response.Body.Close()

	reader := bufio.NewReader(response.Body)
	if line, err := reader.ReadString('\n'); err != nil || line != ": connected\n" {
		t.Fatalf("first line = %q, %v; want connected comment", line, err)
	}
	if line, err := reader.ReadString('\n'); err != nil || line != "\n" {
		t.Fatalf("second line = %q, %v; want blank comment terminator", line, err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read delayed SSE line: %v", err)
		}
		if line == "event: keepalive\n" {
			return
		}
	}
	t.Fatal("timed out waiting for delayed keepalive")
}
