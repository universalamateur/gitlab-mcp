package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
)

func TestRootCommand(t *testing.T) {
	const port = "8080"

	baseURL := "http://localhost:" + port
	glMux := http.NewServeMux()

	glServer := httptest.NewServer(glMux)
	defer glServer.Close()

	authServerMetadata := auth.AuthorizationServer{
		Issuer:                glServer.URL,
		AuthEndpoint:          glServer.URL + "/oauth/authorize",
		TokenEndpoint:         glServer.URL + "/oauth/token",
		IntrospectionEndpoint: glServer.URL + "/oauth/introspect",
	}

	glMux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(authServerMetadata); err != nil {
			http.Error(w, fmt.Sprintf("json.Encode: %v", err), http.StatusInternalServerError)
		}
	})

	glMux.HandleFunc("/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("ParseForm(): %v", err), http.StatusBadRequest)

			return
		}

		if got, want := r.Form.Get("token"), "test-bearer-token"; got != want {
			http.Error(w, fmt.Sprintf(`Form.Get("token") = %q, want %q`, got, want), http.StatusBadRequest)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"active": true, "aud": [%q], "exp": 0}`, baseURL+"/mcp")
	})

	testCases := []struct {
		name       string
		request    *http.Request
		wantStatus int
		wantHeader http.Header
		wantBody   string
	}{
		{
			name:       "Unauthorized",
			request:    mustNewRequest(t, http.MethodGet, baseURL+"/mcp", nil, ""),
			wantStatus: http.StatusUnauthorized,
			wantHeader: http.Header{
				"WWW-Authenticate": []string{`Bearer realm="` + baseURL + `/mcp", resource_server_metadata_url="` + baseURL + `/.well-known/oauth-protected-resource/mcp"`},
			},
		},
		{
			name:       "Resource Metadata",
			request:    mustNewRequest(t, http.MethodGet, baseURL+"/.well-known/oauth-protected-resource/mcp", nil, ""),
			wantStatus: http.StatusOK,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBody: mustEncode(auth.ProtectedResource{
				Resource: baseURL + "/mcp",
				AuthorizationServers: []auth.AuthorizationServer{
					authServerMetadata,
				},
			}),
		},
		{
			name: "With Authorization header",
			request: mustNewRequest(t, http.MethodGet, baseURL+"/mcp", map[string]string{
				"Authorization": `Bearer test-bearer-token`,
			}, ""),
			wantStatus: http.StatusAccepted,
		},
		{
			name: "With Private-Token header",
			request: mustNewRequest(t, http.MethodGet, baseURL+"/mcp", map[string]string{
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
		"--base-url", glServer.URL,
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

			for name, want := range tc.wantHeader {
				got := resp.Header.Get(name)
				if got != want[0] {
					t.Errorf("resp.Header.Get(%q) = %q, want %q", name, got, want[0])
				}
			}

			if tc.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(tc.wantBody, strings.TrimSpace(string(body))); diff != "" {
					t.Errorf("Request body differs (-want/+got):\n%s", diff)
				}
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

func mustEncode(a any) string {
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}

	return string(b)
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

func TestResourceURL(t *testing.T) {
	testCases := []struct {
		listenAddr string
		want       string
	}{
		{":8080", "http://localhost:8080/mcp"},
		{"localhost", "https://localhost/mcp"},
		{"localhost:8080", "http://localhost:8080/mcp"},
		{"localhost:443", "https://localhost/mcp"},
		{"mcp.example.com:8080", "https://mcp.example.com:8080/mcp"},
		{"mcp.example.com:443", "https://mcp.example.com/mcp"},
	}

	for _, tc := range testCases {
		t.Run(tc.listenAddr, func(t *testing.T) {
			if got := resourceURL(tc.listenAddr).String(); got != tc.want {
				t.Errorf("resourceURL(%q) = %q, want %q", tc.listenAddr, got, tc.want)
			}
		})
	}
}
