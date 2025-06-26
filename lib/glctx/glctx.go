// Package glctx provides utilities for storing and retrieving GitLab API clients
// from Go contexts. This allows GitLab clients to be passed through request
// processing pipelines without explicit parameter passing.
package glctx

import (
	"context"
	"errors"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ErrNoClient is the error returned when no GitLab client is found in the context.
var ErrNoClient = errors.New("gitlab client not found in context")

type key struct{}

// clientKey is a singleton instance used as the context key for GitLab clients.
var clientKey = &key{}

// clientErrorKey is a singleton instance used as the context key for errors.
var clientErrorKey = &key{}

// WithClient returns a new context that carries the provided GitLab client or error.
// The client can later be retrieved using the Client function.
func WithClient(ctx context.Context, client *gitlab.Client) context.Context {
	return context.WithValue(ctx, clientKey, client)
}

// WithError returns a new context that carries the provided error.
// The error can later be retrieved using the Client function.
func WithClientError(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, clientErrorKey, err)
}

// Client retrieves the GitLab client from the provided context.
// It returns the client and an error indicating whether a valid client was found.
// If an error is set on the context with WithError, it returns that error instead of the client.
// If no client exists in the context or the client is nil, it returns ErrNoClient.
func Client(ctx context.Context) (*gitlab.Client, error) {
	err, _ := ctx.Value(clientErrorKey).(error)
	if err != nil {
		return nil, err
	}

	c, ok := ctx.Value(clientKey).(*gitlab.Client)
	if !ok || c == nil {
		return nil, ErrNoClient
	}

	return c, nil
}
