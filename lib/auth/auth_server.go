package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/elnormous/contenttype"
)

var (
	ErrBadResponse           = errors.New("bad response")
	ErrTokenNotActive        = errors.New("token is not active")
	ErrTokenAudienceMismatch = errors.New("token audience does not match protected resource")
	ErrTokenExpired          = errors.New("token is expired")
)

// AuthorizationServer represents an OAuth2 authorization server, as specified in RFC 8414.
type AuthorizationServer struct {
	Issuer                string `json:"issuer,omitempty"`
	AuthEndpoint          string `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string `json:"token_endpoint,omitempty"`
	IntrospectionEndpoint string `json:"introspection_endpoint,omitempty"`
}

// LoadAuthorizationServer fetches the authorization server's metadata from the specified issuer URL.
func LoadAuthorizationServer(ctx context.Context, issuer string) (*AuthorizationServer, error) {
	const authServerMetadataPath = "/.well-known/oauth-authorization-server"

	u, err := url.Parse(issuer)
	if err != nil {
		return nil, fmt.Errorf("url.Parse(%q): %w", issuer, err)
	}

	u.Path = path.Join(authServerMetadataPath, u.Path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.DefaultClient.Do(%q): %w", u.String(), err)
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: http.Get(%q): returned status code %d", ErrBadResponse, u.String(), resp.StatusCode)
	}

	contentType, err := contenttype.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("contenttype.ParseMediaType: %w", err)
	}

	if got, want := contentType.MIME(), "application/json"; got != want {
		return nil, fmt.Errorf("%w: authorization server endpoint returned content type %q, want %q", ErrBadResponse, got, want)
	}

	var authServer AuthorizationServer
	if err := json.NewDecoder(resp.Body).Decode(&authServer); err != nil {
		return nil, fmt.Errorf("json.Decode: %w", err)
	}

	return &authServer, nil
}

func (s AuthorizationServer) Introspect(ctx context.Context, clientID, token string) (*Introspection, error) {
	body := url.Values{
		"token":           []string{token},
		"token_type_hint": []string{"access_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.IntrospectionEndpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext:	%w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: introspection endpoint returned status code %d", ErrBadResponse, resp.StatusCode)
	}

	contentType, err := contenttype.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("contenttype.ParseMediaType: %w", err)
	}

	if got, want := contentType.MIME(), "application/json"; got != want {
		return nil, fmt.Errorf("%w: introspection endpoint returned content type %q, want %q", ErrBadResponse, got, want)
	}

	var intro Introspection
	if err := json.NewDecoder(resp.Body).Decode(&intro); err != nil {
		return nil, fmt.Errorf("json.Decode: %w", err)
	}

	return &intro, nil
}

type Introspection struct {
	Active    bool     `json:"active"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
}

func (res Introspection) Verify(audience string) error {
	if !res.Active {
		return ErrTokenNotActive
	}

	if len(res.Audience) != 0 && !slices.Contains(res.Audience, audience) {
		return fmt.Errorf("%w: got %q, want %q", ErrTokenAudienceMismatch, res.Audience, audience)
	}

	if res.ExpiresAt != 0 {
		t := time.Unix(res.ExpiresAt, 0)
		if t.Before(time.Now()) {
			return ErrTokenExpired
		}
	}

	return nil
}
