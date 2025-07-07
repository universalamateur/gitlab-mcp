package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const (
	defaultIssuer = "https://gitlab.com"
)

var wellKnownClientIDs = map[string]string{
	defaultIssuer:            "039e1e052b348d4e45e55bd50bd5cd1fa28b6ee3d044962587fd78bf68ceae4f",
	"https://ops.gitlab.net": "31cdb7e8369904a9c45ae5491cb5e1c6152b2080b888a7b427256e34add04e58",
}

var (
	ErrTokenNotFound = errors.New("OAuth token not found")
	ErrBadConfig     = errors.New("bad config")
)

// ClientID returns the client ID for the given issuer.
// If clientID is not empty, it is returned.
// If issuer is empty, it defaults to defaultIssuer.
// If issuer is unknown, ErrBadConfig is returned.
func ClientID(clientID, issuer string) (string, error) {
	if clientID != "" {
		return clientID, nil
	}

	if issuer == "" {
		issuer = defaultIssuer
	}

	id, ok := wellKnownClientIDs[issuer]
	if !ok {
		return "", fmt.Errorf("%w: OAuth client ID not found for issuer %q; use the `--client-id` flag to provide a custom client ID", ErrBadConfig, issuer)
	}

	return id, nil
}

// Config creates an OAuth2 config for the GitLab instance at baseURL.
// If baseURL is empty, it defaults to gitlab.com.
func Config(baseURL, clientID string) (*oauth2.Config, error) {
	clientID, err := ClientID(clientID, baseURL)
	if err != nil {
		return nil, err
	}

	return &oauth2.Config{
		ClientID: clientID,
		Endpoint: endpoint(baseURL),
		Scopes:   []string{"api"},
	}, nil
}

func endpoint(baseURL string) oauth2.Endpoint {
	if baseURL == "" {
		return endpoints.GitLab
	}

	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/api/v4")

	return oauth2.Endpoint{
		AuthURL:       baseURL + "/oauth/authorize",
		TokenURL:      baseURL + "/oauth/token",
		DeviceAuthURL: baseURL + "/oauth/authorize_device",
	}
}

// TokenPath returns the path of the token file based on the operating system:
//
// - Mac OS: `~/Library/Application Support/gitlab-mcp/token.json`
// - Linux: `~/.config/gitlab-mcp/token.json`
// - Windows: `%APPDATA%\gitlab-mcp\token.json`
// - Other: `~/.gitlab-mcp/token.json`
//
//nolint:godot
func TokenPath() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "gitlab-mcp", "token.json")
	case "linux":
		return filepath.Join(os.Getenv("HOME"), ".config", "gitlab-mcp", "token.json")
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "gitlab-mcp", "token.json")
	default:
		return filepath.Join(os.Getenv("HOME"), ".gitlab-mcp", "token.json")
	}
}

// FileTokenSource returns an oauth2.TokenSource that is backed by a file on the file system.
// The token is refreshed automatically. New tokens are automatically written to the file.
//
//nolint:ireturn // Return type is determined by oauth2.ReuseTokenSource.
func FileTokenSource(ctx context.Context, issuer, filename string, config *oauth2.Config) oauth2.TokenSource {
	if issuer == "" {
		issuer = defaultIssuer
	}

	src := &fileBackedToken{
		ctx:        ctx,
		issuer:     issuer,
		filename:   filename,
		config:     config,
		fileSystem: osFS{},
	}

	return oauth2.ReuseTokenSource(nil, src)
}

//nolint:containedctx // We have to implement the TokenSource interface, which does not accept a context.
type fileBackedToken struct {
	ctx      context.Context
	issuer   string
	filename string
	config   interface {
		TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource
	}
	fileSystem fileSystem
	mu         sync.Mutex
}

func (t *fileBackedToken) Token() (*oauth2.Token, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	storedToken, err := t.load()
	if err != nil {
		return nil, err
	}

	refreshedToken, err := t.config.TokenSource(t.ctx, storedToken).Token()
	if err != nil {
		return nil, fmt.Errorf("config.TokenSource(): %w", err)
	}

	if storedToken.AccessToken != refreshedToken.AccessToken ||
		storedToken.RefreshToken != refreshedToken.RefreshToken {
		if err := StoreToken(t.filename, t.issuer, refreshedToken, withFileSystem(t.fileSystem)); err != nil {
			return nil, fmt.Errorf("StoreToken(filename=%q, issuer=%q): %w", t.filename, t.issuer, err)
		}
	}

	return refreshedToken, nil
}

func (t *fileBackedToken) load() (*oauth2.Token, error) {
	data, err := t.fileSystem.ReadFile(t.filename)
	if err != nil {
		return nil, fmt.Errorf("ReadFile(%q): %w", t.filename, err)
	}

	stored, err := parseStoredTokens(data)
	if err != nil {
		return nil, fmt.Errorf("parseStoredTokens(): %w", err)
	}

	token, ok := stored.Tokens[t.issuer]
	if !ok {
		return nil, fmt.Errorf("%w for issuer %q; run `gitlab-mcp auth` to generate a token", ErrTokenNotFound, t.issuer)
	}

	return token, nil
}

func parseStoredTokens(data []byte) (storedTokens, error) {
	stored, err := parseLegacyToken(data)
	if !errors.Is(err, errNotLegacyToken) {
		return stored, err
	}

	if err := json.Unmarshal(data, &stored); err != nil {
		return storedTokens{}, fmt.Errorf("json.Unmarshal(): %w", err)
	}

	return stored, nil
}

var errNotLegacyToken = errors.New("not a legacy token")

// parseLegacyToken parses data directly as an *oauth2.Token and converts it to the storedTokens format.
// If data does not represent an *oauth2.Token, it returns a errNotLegacyToken error.
func parseLegacyToken(data []byte) (storedTokens, error) {
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return storedTokens{}, fmt.Errorf("json.Unmarshal legacy token: %w", err)
	}

	if token.AccessToken == "" {
		return storedTokens{}, errNotLegacyToken
	}

	// Convert legacy single token to new format with default issuer
	stored := storedTokens{
		Tokens: map[string]*oauth2.Token{
			defaultIssuer: &token,
		},
	}

	return stored, nil
}

type storedTokens struct {
	Tokens map[string]*oauth2.Token `json:"tokens"`
}

type storeTokenOptions struct {
	fs fileSystem
}

type storeTokenOption func(*storeTokenOptions)

func withFileSystem(fs fileSystem) storeTokenOption {
	return func(opts *storeTokenOptions) {
		opts.fs = fs
	}
}

// StoreToken writes the provided token to the specified file.
func StoreToken(filename, issuer string, token *oauth2.Token, options ...storeTokenOption) error {
	var (
		opts = storeTokenOptions{
			fs: osFS{},
		}
		stored storedTokens
	)

	for _, opt := range options {
		opt(&opts)
	}

	data, err := opts.fs.ReadFile(filename)

	switch {
	case errors.Is(err, os.ErrNotExist):
		// no-op
	case err != nil:
		return fmt.Errorf("ReadFile(%q): %w", filename, err)
	default:
		stored, err = parseStoredTokens(data)
		if err != nil {
			return fmt.Errorf("parseStoredTokens(): %w", err)
		}
	}

	if stored.Tokens == nil {
		stored.Tokens = make(map[string]*oauth2.Token)
	}

	stored.Tokens[issuer] = token

	data, err = json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("json.Marshal(): %w", err)
	}

	if err := opts.fs.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("WriteFile(%q): %w", filename, err)
	}

	return nil
}

// fileSystem is an interface that abstracts file system operations for testing.
type fileSystem interface {
	ReadFile(filename string) ([]byte, error)
	WriteFile(filename string, data []byte, perm fs.FileMode) error
}

// osFS implements FileSystem using the os package.
type osFS struct{}

func (fs osFS) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename) //nolint:wrapcheck
}

func (fs osFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(filename)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("os.MkdirAll(%q): %w", dir, err)
	}

	if info, err := os.Stat(filename); err == nil && info.Mode().Perm() != perm {
		if err := os.Chmod(filename, perm); err != nil {
			return fmt.Errorf("os.Chmod(%q): %w", filename, err)
		}
	}

	return os.WriteFile(filename, data, perm) //nolint:wrapcheck
}

// Wrap wraps an HTTP handler to check for the presence of a "Private-Token" header.
// If the header is present, the request is passed to the wrapped handler.
// Otherwise, a 401 Unauthorized status is returned.
func Wrap(hndl http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Private-Token") != "" {
			hndl.ServeHTTP(w, req)
			return
		}

		http.Error(w, "Unauthorized. Please authenticate with a valid Private-Token header.", http.StatusUnauthorized)
	})
}
