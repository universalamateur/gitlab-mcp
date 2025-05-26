// Package glctx provides utilities for storing and retrieving GitLab API clients
// from Go contexts. This allows GitLab clients to be passed through request
// processing pipelines without explicit parameter passing.
package glctx

import (
	"context"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type key struct{}

// clientKey is a singleton instance used as the context key for GitLab clients.
var clientKey = &key{}

// WithClient returns a new context that carries the provided GitLab client.
// The client can later be retrieved using the Client function.
func WithClient(ctx context.Context, client *gitlab.Client) context.Context {
	return context.WithValue(ctx, clientKey, client)
}

// Client retrieves the GitLab client from the provided context.
// It returns the client and a boolean indicating whether a valid client was found.
// If no client exists in the context or the client is nil, it returns (nil, false).
func Client(ctx context.Context) (*gitlab.Client, bool) {
	client, ok := ctx.Value(clientKey).(*gitlab.Client)
	return client, ok && client != nil
}
