package glclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"

	"gitlab.com/fforster/gitlab-mcp/lib/auth"
	"gitlab.com/fforster/gitlab-mcp/lib/build"
)

const defaultBaseURL = "https://gitlab.com"

var (
	// ErrInvalidScheme is returned when GitLab URL doesn't use HTTPS.
	ErrInvalidScheme = errors.New("GITLAB_URL must use https scheme")
	// ErrConfig is returned when configuration is invalid.
	ErrConfig = errors.New("invalid OAuth configuration")
	// ErrFlagNotFound is returned when a flag is not found.
	ErrFlagNotFound = errors.New("flag not found")
)

// DefaultBaseURL returns the default GitLab URL or the URL specified by the GITLAB_URL environment variable.
func DefaultBaseURL() string {
	if baseURL, ok := os.LookupEnv("GITLAB_URL"); ok {
		return baseURL
	}

	return defaultBaseURL
}

type Config struct {
	AuthSource gitlab.AuthSource
	BaseURL    string // Optional
	ClientID   string

	AuthOAuth bool
	AuthPAT   bool
}

// ConfigFromFlags creates a new configuration by based on the provided flag set.
// GitLab tokens are determined from the environment or file system.
func ConfigFromFlags(ctx context.Context, fs *pflag.FlagSet) (*Config, error) {
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

	authOAuth, err := fs.GetBool("oauth")
	if err != nil {
		authOAuth = true
	}

	authPAT, err := fs.GetBool("pat")
	if err != nil {
		authPAT = true
	}

	cfg := &Config{
		BaseURL:  baseURL,
		ClientID: clientID,

		AuthOAuth: authOAuth,
		AuthPAT:   authPAT,
	}

	if err := cfg.validateBaseURL(); err != nil {
		return nil, err
	}

	if err := cfg.configureToken(ctx); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) configureToken(ctx context.Context) error {
	oauthErr := c.configureOAuthToken(ctx)
	if oauthErr == nil {
		return nil
	}

	patErr := c.configurePAT()
	if patErr == nil {
		return nil
	}

	return errors.Join(oauthErr, patErr)
}

func (c *Config) configureOAuthToken(ctx context.Context) error {
	if !c.AuthOAuth {
		return fmt.Errorf("%w: OAuth token disabled", ErrConfig)
	}

	tokenFile := auth.TokenPath()

	if s, err := os.Stat(tokenFile); err != nil || !s.Mode().IsRegular() {
		return fmt.Errorf("%w: token file %q does not exist or is not a regular file", auth.ErrTokenNotFound, tokenFile)
	}

	authConfig, err := auth.Config(c.BaseURL, c.ClientID)
	if err != nil {
		return fmt.Errorf("auth.Config(%q, %q): %w", c.BaseURL, c.ClientID, err)
	}

	c.AuthSource = gitlab.OAuthTokenSource{
		TokenSource: auth.FileTokenSource(ctx, c.BaseURL, tokenFile, authConfig),
	}

	return nil
}

func (c *Config) configurePAT() error {
	if !c.AuthPAT {
		return fmt.Errorf("%w: Personal Access Token disabled", ErrConfig)
	}

	pat, ok := os.LookupEnv("GITLAB_TOKEN")
	if !ok {
		return fmt.Errorf("%w: GITLAB_TOKEN environment variable not set", auth.ErrTokenNotFound)
	}

	c.AuthSource = gitlab.AccessTokenAuthSource{
		Token: pat,
	}

	return nil
}

// ServerConfig represents the server-side configuration.
type ServerConfig struct {
	BaseURL string

	AuthOAuth bool
	AuthPAT   bool
}

// ConfigFromRequest reads configuration from an HTTP request.
// It looks for the 'Private-Token' headers to set the access token.
func ConfigFromRequest(r *http.Request, serverConfig ServerConfig) (*Config, error) {
	clientConfig := &Config{
		BaseURL: serverConfig.BaseURL,

		AuthOAuth: serverConfig.AuthOAuth,
		AuthPAT:   serverConfig.AuthPAT,
	}

	var (
		authorization = r.Header.Get("Authorization")
		privateToken  = r.Header.Get("Private-Token")
	)

	switch {
	case authorization != "":
		clientConfig.AuthSource = gitlab.OAuthTokenSource{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: strings.TrimPrefix(authorization, "Bearer "),
			}),
		}

	case privateToken != "":
		clientConfig.AuthSource = gitlab.AccessTokenAuthSource{
			Token: privateToken,
		}

	default:
		return nil, fmt.Errorf("%w; set the `Authorization` or `Private-Token` header", auth.ErrTokenNotFound)
	}

	err := clientConfig.validateBaseURL()
	if err != nil {
		return nil, err
	}

	return clientConfig, nil
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
func New(cfg *Config) (*gitlab.Client, error) {
	var clientOptions []gitlab.ClientOptionFunc

	if cfg.BaseURL != "" {
		clientOptions = append(clientOptions, gitlab.WithBaseURL(cfg.BaseURL))
	}

	client, err := gitlab.NewAuthSourceClient(cfg.AuthSource, clientOptions...)
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
	s := (*url.URL)(f).String()
	if s == "" {
		return DefaultBaseURL()
	}

	return s
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
