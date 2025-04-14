FROM scratch
MAINTAINER  Florian Forster <fforster@gitlab.com>

# The binary is built by goreleaser outside of the Dockerfile.
COPY gitlab-mcp /usr/bin/gitlab-mcp

ENTRYPOINT ["/usr/bin/gitlab-mcp"]
