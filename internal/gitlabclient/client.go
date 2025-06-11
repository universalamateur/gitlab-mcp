package gitlabclient

import (
	"errors"
	"fmt"
	"net/url"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/internal/config"
)

var (
	// ErrInvalidScheme is returned when GitLab URL doesn't use HTTPS.
	ErrInvalidScheme = errors.New("GITLAB_URL must use https scheme")
)

// New creates a GitLab client with proper validation and security checks.
func New(cfg *config.Config) (*gitlab.Client, error) {
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
