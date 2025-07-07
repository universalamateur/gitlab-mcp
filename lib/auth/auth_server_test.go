package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLoadAuthorizationServer(t *testing.T) {
	cases := []struct {
		name            string
		pathPrefix      string
		content         string
		handler         func(http.ResponseWriter, *http.Request)
		want            AuthorizationServer
		wantErrContains string
	}{
		{
			name: "success",
			content: `{
			  "issuer": "${BASE_URL}",
			  "authorization_endpoint": "${BASE_URL}/oauth/authorize",
			  "token_endpoint": "${BASE_URL}/oauth/token",
			  "introspection_endpoint": "${BASE_URL}/oauth/introspect"
			}`,
			want: AuthorizationServer{
				Issuer:                "${BASE_URL}",
				AuthEndpoint:          "${BASE_URL}/oauth/authorize",
				TokenEndpoint:         "${BASE_URL}/oauth/token",
				IntrospectionEndpoint: "${BASE_URL}/oauth/introspect",
			},
		},
		{
			name:       "with path prefix",
			pathPrefix: "/prefix",
			content: `{
			  "issuer": "${BASE_URL}/prefix",
			  "authorization_endpoint": "${BASE_URL}/prefix/oauth/authorize",
			  "token_endpoint": "${BASE_URL}/prefix/oauth/token",
			  "introspection_endpoint": "${BASE_URL}/prefix/oauth/introspect"
			}`,
			want: AuthorizationServer{
				Issuer:                "${BASE_URL}/prefix",
				AuthEndpoint:          "${BASE_URL}/prefix/oauth/authorize",
				TokenEndpoint:         "${BASE_URL}/prefix/oauth/token",
				IntrospectionEndpoint: "${BASE_URL}/prefix/oauth/introspect",
			},
		},
		{
			name: "endpoint failure",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			},
			wantErrContains: "500",
		},
		{
			name: "unexpected content type",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprint(w, "not json")
			},
			wantErrContains: "text/plain",
		},
		{
			name:            "invalid JSON",
			content:         "!JSON",
			wantErrContains: "json.Decode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()

			srv := httptest.NewServer(mux)
			defer srv.Close()

			baseURL := srv.URL + tc.pathPrefix
			content := strings.ReplaceAll(tc.content, "${BASE_URL}", baseURL)

			mux.HandleFunc("/.well-known/oauth-authorization-server"+tc.pathPrefix, func(w http.ResponseWriter, r *http.Request) {
				if tc.handler != nil {
					tc.handler(w, r)

					return
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")

				fmt.Fprint(w, content)
			})

			got, err := LoadAuthorizationServer(t.Context(), baseURL)

			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr != wantErr {
				t.Errorf("LoadAuthorizationServer(%q): got error %v, want error %v", baseURL, err, wantErr)
			}

			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr && wantErr && !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("LoadAuthorizationServer(%q): got error %v, want error containing %q", baseURL, err, tc.wantErrContains)
			}

			if err != nil {
				return
			}

			want := withBaseURL(tc.want, baseURL)

			if diff := cmp.Diff(&want, got); diff != "" {
				t.Errorf("LoadAuthorizationServer(%q) differs (-want/+got):\n%s", baseURL, diff)
			}
		})
	}
}

func withBaseURL(server AuthorizationServer, baseURL string) AuthorizationServer {
	server.Issuer = strings.ReplaceAll(server.Issuer, "${BASE_URL}", baseURL)
	server.AuthEndpoint = strings.ReplaceAll(server.AuthEndpoint, "${BASE_URL}", baseURL)
	server.TokenEndpoint = strings.ReplaceAll(server.TokenEndpoint, "${BASE_URL}", baseURL)
	server.IntrospectionEndpoint = strings.ReplaceAll(server.IntrospectionEndpoint, "${BASE_URL}", baseURL)

	return server
}

func TestIntrospect(t *testing.T) {
	cases := []struct {
		name            string
		issuer          string
		clientID        string
		token           string
		handler         func(http.ResponseWriter, *http.Request)
		want            Introspection
		wantErrContains string
	}{
		{
			name:     "success",
			issuer:   "https://gitlab.example.com",
			clientID: "test-client-id",
			token:    "test-token",
			want: Introspection{
				Active:    true,
				Audience:  []string{"audience-1", "audience-2"},
				ExpiresAt: 1751641681,
			},
		},
		{
			name:     "server error",
			issuer:   "https://gitlab.example.com",
			clientID: "test-client-id",
			token:    "test-token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			},
			wantErrContains: "500",
		},
		{
			name:     "unexpected content type",
			issuer:   "https://gitlab.example.com",
			clientID: "test-client-id",
			token:    "test-token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprint(w, "not json")
			},
			wantErrContains: "text/plain",
		},
		{
			name:     "invalid JSON",
			issuer:   "https://gitlab.example.com",
			clientID: "test-client-id",
			token:    "test-token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, "!JSON")
			},
			wantErrContains: "json.Decode",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var handler http.Handler = newIntrospectionHandler(tc.clientID, map[string]Introspection{
				tc.token: tc.want,
			})

			if tc.handler != nil {
				handler = http.HandlerFunc(tc.handler)
			}

			mux := http.NewServeMux()
			mux.Handle("/oauth/introspect", handler)

			srv := httptest.NewServer(mux)
			defer srv.Close()

			authServer := AuthorizationServer{
				Issuer:                srv.URL,
				IntrospectionEndpoint: srv.URL + "/oauth/introspect",
			}

			got, err := authServer.Introspect(t.Context(), tc.clientID, tc.token)

			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr != wantErr {
				t.Errorf("Introspect(%q, %q): got error %v, want error %v", tc.clientID, tc.token, err, wantErr)
			}

			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr && wantErr && !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Introspect(%q, %q): got error %v, want error containing %q", tc.clientID, tc.token, err, tc.wantErrContains)
			}

			if err != nil {
				return
			}

			if diff := cmp.Diff(&tc.want, got); diff != "" {
				t.Errorf("Introspect(%q, %q) differs (-want/+got):\n%s", tc.clientID, tc.token, diff)
			}
		})
	}
}

type fakeIntrospectionHandler struct {
	clientID string
	tokens   map[string]Introspection
}

func newIntrospectionHandler(clientID string, tokens map[string]Introspection) *fakeIntrospectionHandler {
	return &fakeIntrospectionHandler{
		clientID: clientID,
		tokens:   tokens,
	}
}

func (h *fakeIntrospectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		http.Error(w, fmt.Sprintf("bad content type: %q", ct), http.StatusBadRequest)
		return
	}

	if got, want := r.Header.Get("Authorization"), basicAuth(h.clientID, ""); got != want {
		http.Error(w, fmt.Sprintf(`Get("Authorization") = %q, want %q`, got, want), http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("ParseForm(): %v", err), http.StatusBadRequest)
		return
	}

	token := r.Form.Get("token")

	resp, ok := h.tokens[token]
	if !ok {
		http.Error(w, fmt.Sprintf(`Get("token") = %q: unexpected token`, token), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("json.NewEncoder: %v", err), http.StatusInternalServerError)
	}
}

func basicAuth(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestIntrospection_Verify(t *testing.T) {
	cases := []struct {
		name            string
		intro           Introspection
		audience        string
		wantErrContains string
	}{
		{
			name: "success",
			intro: Introspection{
				Active:    true,
				Audience:  []string{"audience-1", "audience-2"},
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
			audience: "audience-1",
		},
		{
			name: "inactive",
			intro: Introspection{
				Active: false,
			},
			wantErrContains: "not active",
		},
		{
			name: "expired",
			intro: Introspection{
				Active:    true,
				Audience:  []string{"audience-1", "audience-2"},
				ExpiresAt: time.Now().Add(-time.Hour).Unix(),
			},
			audience:        "audience-2",
			wantErrContains: "expired",
		},
		{
			name: "audience mismatch",
			intro: Introspection{
				Active:    true,
				Audience:  []string{"audience-1"},
				ExpiresAt: 1751641681,
			},
			audience:        "bad-audience",
			wantErrContains: `"bad-audience"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.intro.Verify(tc.audience)

			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr != wantErr {
				t.Errorf("Verify(): got error %v, want error %v", err, wantErr)
			}

			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr && wantErr && !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Verify(): got error %v, want error containing %q", err, tc.wantErrContains)
			}
		})
	}
}
