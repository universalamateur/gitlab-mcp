package cmd

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRootCommand(t *testing.T) {
	const port = "8080"

	testCases := []struct {
		name       string
		request    *http.Request
		wantStatus int
	}{
		{
			name:       "Unauthorized",
			request:    mustNewRequest(t, http.MethodGet, "http://localhost:"+port+"/mcp", nil, ""),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "With Private-Token header",
			request: mustNewRequest(t, http.MethodGet, "http://localhost:"+port+"/mcp", map[string]string{
				"Private-Token": `test-private-token`,
			}, ``),
			wantStatus: http.StatusAccepted,
		},
	}

	// Start MCP server in a Goroutine.
	cmd := New()

	err := cmd.ParseFlags([]string{
		"--transport", "http",
		"--listen-addr", "localhost:" + port,
		"--base-url", "https://gitlab.example.com",
		"--client-id", "test-client-id",
	})
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- cmd.ExecuteContext(t.Context())
	}()

	waitForServer(t, "localhost:"+port)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.DefaultClient.Do(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("resp.StatusCode = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func mustNewRequest(t *testing.T, method, url string, header map[string]string, body string) *http.Request {
	t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(t.Context(), method, url, r)
	if err != nil {
		t.Fatalf("http.NewRequest(%q, %q): %v", method, url, err)
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	return req
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	t.Logf("Waiting for %q to become available...", addr)

	var (
		start   = time.Now()
		backoff = 10 * time.Millisecond
	)

	for {
		if deadline, ok := t.Deadline(); ok && time.Now().After(deadline) {
			t.Fatalf("Timed out waiting for %q to become available", addr)
		}

		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			t.Logf("%q became available after %s", addr, time.Since(start))

			return
		}

		time.Sleep(backoff)

		backoff = min(2*backoff, time.Second)
	}
}
