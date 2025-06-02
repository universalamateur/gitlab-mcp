package glclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

var (
	// ErrMissingToken is returned when GITLAB_TOKEN environment variable is not set.
	ErrMissingToken = errors.New("GITLAB_TOKEN environment variable is not set")
	// ErrInvalidScheme is returned when GitLab URL doesn't use HTTPS.
	ErrInvalidScheme = errors.New("GITLAB_URL must use https scheme")
)

type Config struct {
	GitLabToken    string
	OAuthTokenFile string
	GitLabURL      string // Optional
}

// NewConfig creates a new configuration by reading environment variables.
func NewConfig() (*Config, error) {
	cfg := &Config{
		GitLabURL: os.Getenv("GITLAB_URL"), // Can be empty
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
	s, err := os.Stat(tokenFile)
	if err == nil && s.Mode().IsRegular() {
		c.OAuthTokenFile = tokenFile
		return nil
	}

	pat, havePAT := os.LookupEnv("GITLAB_TOKEN")
	if havePAT {
		c.GitLabToken = pat
		return nil
	}

	return fmt.Errorf("no authentication token available. Run `gitlab-mcp auth` to generate a token")
}

func (c *Config) validateBaseURL() error {
	if c.GitLabURL == "" {
		return nil
	}

	u, err := url.ParseRequestURI(c.GitLabURL)
	if err != nil {
		return fmt.Errorf("url.ParseRequestURI(%q): %w", c.GitLabURL, err)
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
	if cfg.GitLabURL != "" {
		clientOptions = append(clientOptions, gitlab.WithBaseURL(cfg.GitLabURL))
	}

	var (
		client *gitlab.Client
		err    error
	)

	switch {
	case cfg.OAuthTokenFile != "":
		as := gitlab.OAuthTokenSource{
			TokenSource: auth.FileTokenSource(ctx, cfg.OAuthTokenFile, auth.Config()),
		}
		client, err = gitlab.NewAuthSourceClient(as)

	case cfg.GitLabToken != "":
		client, err = gitlab.NewClient(cfg.GitLabToken, clientOptions...)

	default:
		return nil, fmt.Errorf("no authentication token available. Run `gitlab-mcp auth` to generate a token")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	client.UserAgent = "gitlab-mcp/" + build.Version()

	return client, nil
}
