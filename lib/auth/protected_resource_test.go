package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMetadataURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource string
		want     string
		wantErr  bool
	}{
		{
			name:     "basic resource URL",
			resource: "https://example.com/api",
			want:     "https://example.com/.well-known/oauth-protected-resource/api",
		},
		{
			name:     "resource URL with path",
			resource: "https://example.com/api/v1/resource",
			want:     "https://example.com/.well-known/oauth-protected-resource/api/v1/resource",
		},
		{
			name:     "resource URL with trailing slash",
			resource: "https://example.com/api/",
			want:     "https://example.com/.well-known/oauth-protected-resource/api",
		},
		{
			name:     "root resource URL",
			resource: "https://example.com",
			want:     "https://example.com/.well-known/oauth-protected-resource",
		},
		{
			name:     "invalid URL",
			resource: "\n",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pr := NewProtectedResource(tc.resource)

			got, err := pr.MetadataURL()

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("MetadataURL() = (%q, %v), want error: %v", got.String(), err, tc.wantErr)
			}

			if err != nil {
				return
			}

			if got.String() != tc.want {
				t.Errorf("MetadataURL() = %q, want %q", got.String(), tc.want)
			}
		})
	}
}

func TestMetadataHandler(t *testing.T) {
	t.Parallel()

	pr := NewProtectedResource("https://example.gitlab.com/mcp",
		WithAuthorizationServers(AuthorizationServer{
			Issuer: "https://example.gitlab.com",
		}),
		WithClientID("test-client-id"),
	)

	mux := http.NewServeMux()
	pr.RegisterMetadataHandler(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	url := srv.URL + "/.well-known/oauth-protected-resource/mcp"
	resp, err := srv.Client().Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	var got ProtectedResource
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if diff := cmp.Diff(pr, &got, cmpopts.IgnoreUnexported(ProtectedResource{})); diff != "" {
		t.Errorf("MetadataHandler differs (-want/+got):\n%s", diff)
	}
}

func TestProtectedResource_Wrap(t *testing.T) {
	cases := []struct {
		name       string
		request    func(baseURL string) *http.Request
		wantStatus int
		wantHeader http.Header
	}{
		{
			name: "private access token",
			request: func(baseURL string) *http.Request {
				r, err := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
				if err != nil {
					panic(err)
				}

				r.Header.Set("Private-Token", "test-token")

				return r
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "OAuth bearer token",
			request: func(baseURL string) *http.Request {
				r, err := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
				if err != nil {
					panic(err)
				}

				r.Header.Set("Authorization", "Bearer test-token")

				return r
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "without authorization",
			request: func(baseURL string) *http.Request {
				r, err := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
				if err != nil {
					panic(err)
				}

				return r
			},
			wantStatus: http.StatusUnauthorized,
			wantHeader: http.Header{
				"Www-Authenticate": {`Bearer realm="https://gitlab.example.com/mcp", resource_server_metadata_url="https://gitlab.example.com/.well-known/oauth-protected-resource/mcp"`},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const clientID = "test-client-id"
			const token = "test-token"

			// OAuth authentication server
			authMux := http.NewServeMux()
			authSrv := httptest.NewServer(authMux)
			defer authSrv.Close()

			authServer := AuthorizationServer{
				Issuer:                authSrv.URL,
				IntrospectionEndpoint: authSrv.URL + "/oauth/introspect",
			}

			authMux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(authServer)
			})

			authMux.Handle("/oauth/introspect", newIntrospectionHandler(clientID, map[string]Introspection{
				token: {
					Active:   true,
					Audience: []string{"https://gitlab.example.com/mcp"},
				},
			}))

			// OAuth resource server
			pr := NewProtectedResource("https://gitlab.example.com/mcp",
				WithAuthorizationServers(authServer),
				WithClientID(clientID),
			)

			resourceHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			resourceMux := http.NewServeMux()
			resourceMux.Handle("/mcp", pr.Wrap(resourceHandler))

			resourceSrv := httptest.NewServer(resourceMux)
			defer resourceSrv.Close()

			// Test request to OAuth protected resource
			req := tc.request(resourceSrv.URL)
			resp, err := resourceSrv.Client().Do(req)
			if err != nil {
				t.Fatalf("Client().Do(): %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Got status %d, want %d", resp.StatusCode, tc.wantStatus)
			}

			for key, wantValues := range tc.wantHeader {
				if got := resp.Header.Get(key); got != wantValues[0] {
					t.Errorf("Header.Get(%q) = %q, want %q", key, got, wantValues[0])
				}
			}
		})
	}
}
