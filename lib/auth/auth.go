package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const ClientID = "039e1e052b348d4e45e55bd50bd5cd1fa28b6ee3d044962587fd78bf68ceae4f"

func Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID: ClientID,
		Endpoint: endpoints.GitLab,
		Scopes:   []string{"api"},
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
		ctx:      ctx,
		filename: filename,
		config:   config,
	}

	return oauth2.ReuseTokenSource(nil, src)
}

//nolint:containedctx // We have to implement the TokenSource interface, which does not accept a context.
type fileBackedToken struct {
	ctx      context.Context
	filename string
	config   *oauth2.Config
	mu       sync.Mutex
}

func (t *fileBackedToken) Token() (*oauth2.Token, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(t.filename)
	if err != nil {
		return nil, fmt.Errorf("io.ReadFile(%q): %w", t.filename, err)
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
		if err := StoreToken(t.filename, refreshedToken); err != nil {
			return nil, fmt.Errorf("writeToken(): %w", err)
		}
	}

	return refreshedToken, nil
}

// StoreToken writes the provided token to the specified file.
func StoreToken(filename string, token *oauth2.Token) error {
	fh, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q): %w", filename, err)
	}
	defer fh.Close()

	if err := json.NewEncoder(fh).Encode(token); err != nil {
		return fmt.Errorf("json.Encoder(): %w", err)
	}

	return nil
}
