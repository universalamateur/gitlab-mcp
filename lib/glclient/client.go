package glclient

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

var (
	// ErrMissingToken is returned when GITLAB_TOKEN environment variable is not set.
	ErrMissingToken = errors.New("GITLAB_TOKEN environment variable is not set")
	// ErrInvalidScheme is returned when GitLab URL doesn't use HTTPS.
	ErrInvalidScheme = errors.New("GITLAB_URL must use https scheme")
)

type Config struct {
	GitLabToken string
	GitLabURL   string // Optional
	UserAgent   string
}

// NewConfig creates a new configuration by reading environment variables.
func NewConfig() (*Config, error) {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return nil, ErrMissingToken
	}

	version := build.Version()
	if version == "" {
		version = "dev"
	}

	cfg := &Config{
		GitLabToken: token,
		GitLabURL:   os.Getenv("GITLAB_URL"), // Can be empty
		UserAgent:   "gitlab-mcp/" + version,
	}

	return cfg, nil
}

// New creates a GitLab client with proper validation and security checks.
func New(cfg *Config) (*gitlab.Client, error) {
	var clientOptions []gitlab.ClientOptionFunc

	// Validate and add custom GitLab URL if provided
	if cfg.GitLabURL != "" {
		parsedURL, err := url.ParseRequestURI(cfg.GitLabURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GITLAB_URL: %w", err)
		}

		// Security: Enforce HTTPS to prevent sending token over cleartext
		if parsedURL.Scheme != "https" {
			return nil, fmt.Errorf("%w, got '%s'", ErrInvalidScheme, parsedURL.Scheme)
		}

		clientOptions = append(clientOptions, gitlab.WithBaseURL(cfg.GitLabURL))
	}

	// Add request options
	clientOptions = append(clientOptions, gitlab.WithRequestOptions(
		gitlab.WithHeader("User-Agent", cfg.UserAgent),
	))

	client, err := gitlab.NewClient(cfg.GitLabToken, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return client, nil
}
