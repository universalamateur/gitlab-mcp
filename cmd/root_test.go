package cmd

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	testCases := []struct {
		name       string
		request    *http.Request
		wantStatus int
	}{
		{
			name:       "Unauthorized",
			request:    mustNewRequest(http.MethodGet, "http://localhost:8080/mcp", nil, ""),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "With Private-Token header",
			request: mustNewRequest(http.MethodGet, "http://localhost:8080/mcp", map[string]string{
				"Private-Token": `test-private-token`,
			}, ``),
			wantStatus: http.StatusAccepted,
		},
	}

	// Start MCP server in a Goroutine.
	cmd := New()

	cmd.ParseFlags([]string{
		"--transport", "http",
		"--listen-addr", "localhost:8080",
		"--base-url", "https://gitlab.example.com",
		"--client-id", "test-client-id",
	})

	errCh := make(chan error, 1)

	go func() {
		errCh <- cmd.ExecuteContext(t.Context())
	}()

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

func mustNewRequest(method, url string, header map[string]string, body string) *http.Request {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, r)
	if err != nil {
		panic(err)
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	return req
}
