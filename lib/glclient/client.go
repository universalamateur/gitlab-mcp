package glclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

var (
	// ErrMissingToken is returned when GITLAB_TOKEN environment variable is not set.
	ErrMissingToken = errors.New("no authentication token available. Run `gitlab-mcp auth` to generate a token")
	// ErrInvalidScheme is returned when GitLab URL doesn't use HTTPS.
	ErrInvalidScheme = errors.New("GITLAB_URL must use https scheme")
)

type Config struct {
	AccessToken    string
	OAuthTokenFile string
	BaseURL        string // Optional
}

// NewConfig creates a new configuration by reading environment variables.
func NewConfig(fs *pflag.FlagSet) (*Config, error) {
	baseURL, err := fs.GetString("base-url")
	if err != nil {
		return nil, fmt.Errorf("determining base URL from flags: %w", err)
	}

	cfg := &Config{
		BaseURL: baseURL,
	}

	if err := cfg.configureToken(); err != nil {
		return nil, err
	}

	if err := cfg.validateBaseURL(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) configureToken() error {
	tokenFile := auth.TokenPath()

	if s, err := os.Stat(tokenFile); err == nil && s.Mode().IsRegular() {
		c.OAuthTokenFile = tokenFile
		return nil
	}

	if pat, havePAT := os.LookupEnv("GITLAB_TOKEN"); havePAT {
		c.AccessToken = pat
		return nil
	}

	return ErrMissingToken
}

func (c *Config) validateBaseURL() error {
	if c.BaseURL == "" {
		return nil
	}

	u, err := url.ParseRequestURI(c.BaseURL)
	if err != nil {
		return fmt.Errorf("url.ParseRequestURI(%q): %w", c.BaseURL, err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("%w, got %q", ErrInvalidScheme, u.Scheme)
	}

	return nil
}

// New creates a GitLab client with proper validation and security checks.
func New(ctx context.Context, cfg *Config) (*gitlab.Client, error) {
	var clientOptions []gitlab.ClientOptionFunc

	// Validate and add custom GitLab URL if provided
	if cfg.BaseURL != "" {
		clientOptions = append(clientOptions, gitlab.WithBaseURL(cfg.BaseURL))
	}

	var (
		client *gitlab.Client
		err    error
	)

	switch {
	case cfg.OAuthTokenFile != "":
		as := gitlab.OAuthTokenSource{
			TokenSource: auth.FileTokenSource(ctx, cfg.OAuthTokenFile, auth.Config(cfg.BaseURL)),
		}
		client, err = gitlab.NewAuthSourceClient(as)

	case cfg.AccessToken != "":
		client, err = gitlab.NewClient(cfg.AccessToken, clientOptions...)

	default:
		return nil, ErrMissingToken
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	client.UserAgent = "gitlab-mcp/" + build.Version()

	return client, nil
}
