package gitlabclient

import (
	"fmt"
	"net/url"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/internal/config"
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
			return nil, fmt.Errorf("invalid scheme for GITLAB_URL: must be https, but got '%s'", parsedURL.Scheme)
		}

		clientOptions = append(clientOptions, gitlab.WithBaseURL(cfg.GitLabURL))
	}

	// Add request options
	clientOptions = append(clientOptions, gitlab.WithRequestOptions(
		gitlab.WithHeader("User-Agent", cfg.UserAgent),
	))

	return gitlab.NewClient(cfg.GitLabToken, clientOptions...)
}