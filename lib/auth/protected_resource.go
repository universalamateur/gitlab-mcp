package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ProtectedResource represents an OAuth2 protected resource, as specified in RFC 9728.
type ProtectedResource struct {
	Resource             string                `json:"resource,omitempty"`
	AuthorizationServers []AuthorizationServer `json:"authorization_servers,omitempty"`

	clientID string
}

type ProtectedResourceOption func(*ProtectedResource)

// WithAuthorizationServers sets the authorization servers for the protected resource.
func WithAuthorizationServers(authServers ...AuthorizationServer) ProtectedResourceOption {
	return func(r *ProtectedResource) {
		r.AuthorizationServers = authServers
	}
}

// WithClientID sets the client ID for the protected resource.
func WithClientID(clientID string) ProtectedResourceOption {
	return func(r *ProtectedResource) {
		r.clientID = clientID
	}
}

// NewProtectedResource creates a new protected resource with the given URL and options.
func NewProtectedResource(resourceURL string, options ...ProtectedResourceOption) *ProtectedResource {
	r := &ProtectedResource{
		Resource: resourceURL,
	}

	for _, opt := range options {
		opt(r)
	}

	return r
}

// RegisterMetadataHandler registers the metadata handler for the protected resource on the provided multiplexer.
// It ensures that the metadata is served at the well-known URL for OAuth2 protected resources.
// If parsing the resource identifier fails, an error is returned.
func (r ProtectedResource) RegisterMetadataHandler(mux *http.ServeMux) error {
	u, err := r.MetadataURL()
	if err != nil {
		return fmt.Errorf("MetadataURL(): %w", err)
	}

	mux.HandleFunc(u.Path, r.MetadataHandler)

	return nil
}

// MetadataURL returns the URL of the protected resource's metadata.
// If parsing the resource identifier fails, an error is returned.
func (r ProtectedResource) MetadataURL() (*url.URL, error) {
	const resourceMetadataPath = "/.well-known/oauth-protected-resource"

	u, err := url.Parse(r.Resource)
	if err != nil {
		return nil, fmt.Errorf("url.Parse(%q): %w", r.Resource, err)
	}

	u.Path = path.Join(resourceMetadataPath, u.Path)

	return u, nil
}

// MetadataHandler is a HTTP handler that serves the protected resource's metadata.
func (r ProtectedResource) MetadataHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method %s not allowed. Use %s instead.", req.Method, http.MethodGet)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(r); err != nil {
		log.Printf("ProtectedResource.MetadataHandler: json.NewEncoder: %v", err)
		http.Error(w, fmt.Sprintf("json.NewEncoder: %v", err), http.StatusInternalServerError)
	}
}

// Wrap wraps the provided HTTP handler with an OAuth2 authorization check.
// If the request does not have a valid Bearer token or Private-Token header, it responds with an Unauthorized error.
// The Unauthorized response includes a WWW-Authenticate header including the URL of the protected resource's metadata.
func (r ProtectedResource) Wrap(hndl http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if token := req.Header.Get("Authorization"); token != "" {
			token = strings.TrimPrefix(token, "Bearer ")
			if err := r.validateToken(req.Context(), token); err != nil {
				w.Header().Add("Debug", fmt.Sprintf("validateToken(%q): %v", token, err))
				http.Error(w, fmt.Sprintf("validateToken: %v", err), http.StatusUnauthorized)

				return
			}

			hndl.ServeHTTP(w, req)

			return
		}

		if req.Header.Get("Private-Token") != "" {
			hndl.ServeHTTP(w, req)

			return
		}

		u, err := r.MetadataURL()
		if err != nil {
			log.Printf("ProtectedResource.Wrap: MetadataURL(): %v", err)
			http.Error(w, fmt.Sprintf("MetadataURL(): %v", err), http.StatusInternalServerError)
		}

		wwwAuth := fmt.Sprintf("Bearer realm=%q, resource_server_metadata_url=%q",
			r.Resource, u.String(),
		)

		w.Header().Set("WWW-Authenticate", wwwAuth)
		w.WriteHeader(http.StatusUnauthorized)

		fmt.Fprintf(w, "Unauthorized. Please authenticate with a valid Bearer token or Private-Token header.")
	})
}

// validateToken validates the provided reference token by calling the introspection endpoint of the
// authorization server.
func (r ProtectedResource) validateToken(ctx context.Context, token string) error {
	if len(r.AuthorizationServers) == 0 {
		return fmt.Errorf("%w: no authorization server defined for protected resource %q", ErrBadConfig, r.Resource)
	}

	authServer := r.AuthorizationServers[0]
	if authServer.IntrospectionEndpoint == "" {
		return fmt.Errorf("%w: introspection endpoint not defined for authorization server %q", ErrBadConfig, authServer.Issuer)
	}

	resp, err := authServer.Introspect(ctx, r.clientID, token)
	if err != nil {
		return fmt.Errorf("authorization server Introspect: %w", err)
	}

	if err := resp.Verify(r.Resource); err != nil {
		return fmt.Errorf("token introspection: %w", err)
	}

	return nil
}
