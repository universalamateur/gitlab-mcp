package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const ClientID = "039e1e052b348d4e45e55bd50bd5cd1fa28b6ee3d044962587fd78bf68ceae4f"

// Config creates an OAuth2 config for the GitLab instance at baseURL.
// If baseURL is empty, it defaults to gitlab.com.
func Config(baseURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: ClientID,
		Endpoint: endpoint(baseURL),
		Scopes:   []string{"api"},
	}
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
func FileTokenSource(ctx context.Context, filename string, config *oauth2.Config) oauth2.TokenSource {
	src := &fileBackedToken{
		ctx:        ctx,
		filename:   filename,
		config:     config,
		fileSystem: osFS{},
	}

	return oauth2.ReuseTokenSource(nil, src)
}

//nolint:containedctx // We have to implement the TokenSource interface, which does not accept a context.
type fileBackedToken struct {
	ctx      context.Context
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

	data, err := t.fileSystem.ReadFile(t.filename)
	if err != nil {
		return nil, fmt.Errorf("ReadFile(%q): %w", t.filename, err)
	}

	var storedToken oauth2.Token
	if err := json.Unmarshal(data, &storedToken); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(): %w", err)
	}

	refreshedToken, err := t.config.TokenSource(t.ctx, &storedToken).Token()
	if err != nil {
		return nil, fmt.Errorf("config.TokenSource(): %w", err)
	}

	if storedToken.RefreshToken != refreshedToken.RefreshToken {
		data, err := json.Marshal(refreshedToken)
		if err != nil {
			return nil, fmt.Errorf("json.Marshal(): %w", err)
		}

		if err := t.fileSystem.WriteFile(t.filename, data, 0600); err != nil {
			return nil, fmt.Errorf("WriteFile(%q): %w", t.filename, err)
		}
	}

	return refreshedToken, nil
}

// StoreToken writes the provided token to the specified file.
func StoreToken(filename string, token *oauth2.Token) error {
	fs := osFS{}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("json.Marshal(): %w", err)
	}

	if err := fs.WriteFile(filename, data, 0600); err != nil {
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
