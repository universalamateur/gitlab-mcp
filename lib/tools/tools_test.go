package tools

import (
	"strconv"
	"testing"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

// pageFromRequestOptions extracts the requested page from a list of request options.
// This is used when testing functions that use `ScanAndCollect` or `ScanAndCollectN` internally.
// In those cases, the requested page is _not_ provided via the `ListOptions` struct but rather via the request options passed to the GitLab API client.
func pageFromRequestOptions(t *testing.T, reqOpts []gitlab.RequestOptionFunc) int64 {
	t.Helper()

	req, err := retryablehttp.NewRequest("GET", "/dummy", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	for _, opt := range reqOpts {
		if opt == nil {
			continue
		}

		if err := opt(req); err != nil {
			t.Fatalf("applying request option: %v", err)
		}
	}

	q := req.URL.Query()
	if !q.Has("page") {
		t.Log("no page in request options, assuming page 1")

		return 1
	}

	rawPage := q.Get("page")

	page, err := strconv.ParseInt(rawPage, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse page %q: %v", rawPage, err)
	}

	return page
}
