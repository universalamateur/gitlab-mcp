package glclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

const defaultBaseURL = "https://gitlab.com"

var (
	// ErrMissingToken is returned when GITLAB_TOKEN environment variable is not set.
	ErrMissingToken = errors.New("no authentication token available. Run `gitlab-mcp auth` to generate a token")
	// ErrInvalidScheme is returned when GitLab URL doesn't use HTTPS.
	ErrInvalidScheme = errors.New("GITLAB_URL must use https scheme")

	// ErrConfig is returned when configuration is invalid.
	ErrConfig = errors.New("invalid configuration")
)

// DefaultBaseURL returns the default GitLab URL or the URL specified by the GITLAB_URL environment variable.
func DefaultBaseURL() string {
	if baseURL, ok := os.LookupEnv("GITLAB_URL"); ok {
		return baseURL
	}

	return defaultBaseURL
}

type Config struct {
	AccessToken    string
	OAuthTokenFile string
	BaseURL        string // Optional
	ClientID       string
}

var ErrFlagNotFound = errors.New("flag not found")

// NewConfig creates a new configuration by reading environment variables.
func NewConfig(fs *pflag.FlagSet) (*Config, error) {
	f := fs.Lookup("base-url")
	if f == nil {
		return nil, fmt.Errorf("%w: --base-url", ErrFlagNotFound)
	}

	baseURL := f.Value.String()
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/api/v4")

	clientID, err := fs.GetString("client-id")
	if err != nil {
		return nil, fmt.Errorf("determining client ID from flags: %w", err)
	}

	if baseURL != "" && clientID == "" {
		return nil, fmt.Errorf("%w: --client-id must be provided with --base-url", ErrConfig)
	}

	cfg := &Config{
		BaseURL:  baseURL,
		ClientID: clientID,
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
			TokenSource: auth.FileTokenSource(ctx, cfg.BaseURL, cfg.OAuthTokenFile, auth.Config(cfg.BaseURL, cfg.ClientID)),
		}
		client, err = gitlab.NewAuthSourceClient(as, clientOptions...)

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

// BaseURLFlag wraps a string to implement the `pflag.Value` interface.
// This type provides an implementation to customize the behavior of `--base-url` in the `cobra.Command` command-line interface.
type BaseURLFlag url.URL

func (f *BaseURLFlag) String() string {
	return (*url.URL)(f).String()
}

// Set sets the value of the flag.
// The value is sanitized by removing any trailing slashes and the `/api/v4` path from the end of the string.
// If the string is empty, it is set to the default base URL.
func (f *BaseURLFlag) Set(s string) error {
	if s == "" {
		s = DefaultBaseURL()
	}

	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("url.Parse: %w", err)
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	u.Path = strings.TrimSuffix(u.Path, "/api/v4")

	*f = BaseURLFlag(*u)

	return nil
}

func (*BaseURLFlag) Type() string {
	return "url"
}
