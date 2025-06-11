package config

import (
	"errors"
	"os"

	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

var (
	// ErrMissingToken is returned when GITLAB_TOKEN environment variable is not set.
	ErrMissingToken = errors.New("GITLAB_TOKEN environment variable is not set")
)

type Config struct {
	GitLabToken string
	GitLabURL   string // Optional
	UserAgent   string
}

// New creates a new configuration by reading environment variables.
func New() (*Config, error) {
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
