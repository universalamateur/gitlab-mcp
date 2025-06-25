package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		wantEndpoint oauth2.Endpoint
	}{
		{
			name:    "Default GitLab.com",
			baseURL: "",
			wantEndpoint: oauth2.Endpoint{
				AuthURL:       "https://gitlab.com/oauth/authorize",
				TokenURL:      "https://gitlab.com/oauth/token",
				DeviceAuthURL: "https://gitlab.com/oauth/authorize_device",
			},
		},
		{
			name:    "Custom GitLab instance",
			baseURL: "https://gitlab.example.com",
			wantEndpoint: oauth2.Endpoint{
				AuthURL:       "https://gitlab.example.com/oauth/authorize",
				TokenURL:      "https://gitlab.example.com/oauth/token",
				DeviceAuthURL: "https://gitlab.example.com/oauth/authorize_device",
			},
		},
		{
			name:    "Custom GitLab instance with trailing slash",
			baseURL: "https://gitlab.example.com/",
			wantEndpoint: oauth2.Endpoint{
				AuthURL:       "https://gitlab.example.com/oauth/authorize",
				TokenURL:      "https://gitlab.example.com/oauth/token",
				DeviceAuthURL: "https://gitlab.example.com/oauth/authorize_device",
			},
		},
		{
			name:    "Custom GitLab instance with API suffix",
			baseURL: "https://gitlab.example.com/api/v4",
			wantEndpoint: oauth2.Endpoint{
				AuthURL:       "https://gitlab.example.com/oauth/authorize",
				TokenURL:      "https://gitlab.example.com/oauth/token",
				DeviceAuthURL: "https://gitlab.example.com/oauth/authorize_device",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := Config(tc.baseURL)

			if diff := cmp.Diff(tc.wantEndpoint, config.Endpoint); diff != "" {
				t.Errorf("Config(%q) endpoint mismatch (-want/+got):\n%s", tc.baseURL, diff)
			}
		})
	}
}

var cmpOpts = []cmp.Option{
	cmpopts.IgnoreUnexported(oauth2.Token{}),
}

func TestFileBackedToken(t *testing.T) {
	// Define test cases
	tests := []struct {
		name            string
		initialToken    *oauth2.Token
		refreshedToken  *oauth2.Token
		tokenSourceErr  error
		modifyFS        func(fs *mockFS, filename string)
		wantErrContains string
		wantFileWrites  int
	}{
		{
			name: "Basic token retrieval and refresh",
			initialToken: &oauth2.Token{
				AccessToken:  "initial-access-token",
				RefreshToken: "initial-refresh-token",
			},
			refreshedToken: &oauth2.Token{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
			},
			wantFileWrites: 1,
		},
		{
			name: "No refresh token change means no file update",
			initialToken: &oauth2.Token{
				AccessToken:  "initial-access-token",
				RefreshToken: "same-refresh-token",
			},
			refreshedToken: &oauth2.Token{
				AccessToken:  "new-access-token",
				RefreshToken: "same-refresh-token", // Same refresh token
			},
			wantFileWrites: 0,
		},
		{
			name: "File doesn't exist",
			modifyFS: func(fs *mockFS, filename string) {
				delete(fs.files, filename)
			},
			wantErrContains: fs.ErrNotExist.Error(),
		},
		{
			name: "Invalid JSON in token file",
			modifyFS: func(fs *mockFS, filename string) {
				fs.files[filename] = []byte("not valid json")
			},
			wantErrContains: "json.Unmarshal",
		},
		{
			name: "TokenSource error",
			initialToken: &oauth2.Token{
				AccessToken:  "initial-access-token",
				RefreshToken: "initial-refresh-token",
			},
			tokenSourceErr:  errors.New("token source error"),
			wantErrContains: "token source error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const filename = "token.json"

			// Setup mock filesystem
			fs, err := newMockFS(filename, tc.initialToken)
			if err != nil {
				t.Fatalf("NewMockFS(): %v", err)
			}

			if tc.modifyFS != nil {
				tc.modifyFS(fs, filename)
			}

			// Setup mock config
			mockCfg := &mockConfig{
				wantToken: tc.initialToken,

				returnToken: tc.refreshedToken,
				returnErr:   tc.tokenSourceErr,
			}

			// Create fileBackedToken
			fileToken := &fileBackedToken{
				ctx:        t.Context(),
				filename:   filename,
				config:     mockCfg,
				fileSystem: fs,
			}

			// Get token
			token, err := fileToken.Token()

			// Check error expectations
			if gotErr, wantErr := err != nil, tc.wantErrContains != ""; gotErr != wantErr {
				t.Errorf("Token(): got error %v, want error %v", err, wantErr)
			}

			if err != nil && tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Token(): got error %v, want error containing %q", err, tc.wantErrContains)
			}

			if err != nil {
				return
			}

			// Verify TokenSource was called
			if mockCfg.callCount != 1 {
				t.Errorf("TokenSource was called %d times, want 1", mockCfg.callCount)
			}

			// Verify token matches expected
			if diff := cmp.Diff(tc.refreshedToken, token, cmpOpts...); diff != "" {
				t.Errorf("Token mismatch (-want/+got):\n%s", diff)
			}

			if fs.writeCount != tc.wantFileWrites {
				t.Errorf("Got %d file write(s), want %d file write(s)", fs.writeCount, tc.wantFileWrites)
			}

			wantFileToken := tc.initialToken
			if tc.wantFileWrites != 0 {
				wantFileToken = tc.refreshedToken
			}

			wantData, err := json.Marshal(wantFileToken)
			if err != nil {
				t.Fatalf("Failed to marshal expected token: %v", err)
			}

			if diff := cmp.Diff(wantData, fs.files[filename]); diff != "" {
				t.Errorf("File %q content mismatch (-want/+got):\n%s", filename, diff)
			}
		})
	}
}

// mockTokenSource is a test implementation of oauth2.TokenSource.
type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}

// mockConfig implements the TokenSource method of oauth2.Config.
type mockConfig struct {
	wantToken *oauth2.Token

	returnToken *oauth2.Token
	returnErr   error

	callCount int
}

//nolint:ireturn
func (m *mockConfig) TokenSource(ctx context.Context, gotToken *oauth2.Token) oauth2.TokenSource {
	m.callCount++

	// Verify the input token matches what we expect
	if m.wantToken != nil {
		if diff := cmp.Diff(m.wantToken, gotToken, cmpOpts...); diff != "" {
			return &mockTokenSource{
				err: fmt.Errorf("TokenSource token argument differs (-want/+got):\n%s", diff),
			}
		}
	}

	return &mockTokenSource{
		token: m.returnToken,
		err:   m.returnErr,
	}
}

// mockFS implements FileSystem for testing.
type mockFS struct {
	files      map[string][]byte
	writeCount int
}

func newMockFS(filename string, token *oauth2.Token) (*mockFS, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token: %w", err)
	}

	return &mockFS{
		files: map[string][]byte{
			filename: data,
		},
	}, nil
}

func (m *mockFS) ReadFile(filename string) ([]byte, error) {
	data, ok := m.files[filename]
	if !ok {
		return nil, fmt.Errorf("%s: %w", filename, fs.ErrNotExist)
	}

	return data, nil
}

func (m *mockFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	m.writeCount++
	m.files[filename] = data

	return nil
}
