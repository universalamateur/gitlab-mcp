# GitLab MCP

A Model Context Protocol (MCP) server for GitLab, enabling Claude to interact directly with your GitLab instance.

> [!warning]
> ⚠️ This is an unofficial project.
> It is not endorsed or supported by GitLab Inc. and is not recommended for use in production environments.

## :speech_balloon: Feedback

Please provide feedback in https://gitlab.com/gitlab-org/ux-research/-/issues/3495, or (if you don't have access) open a new issue at https://gitlab.com/fforster/gitlab-mcp/-/issues.

## Overview

GitLab MCP implements the [Model Context Protocol](https://github.com/anthropics/model-context-protocol) for GitLab, allowing AI assistants like Claude to access and manipulate GitLab resources. This integration enables Claude to work with:

- Discussions and Notes
- Epics
- Issues
- Jobs
- Merge Requests
- Repository Files and Directories
- Snippets
- User information

## Installation

### Option 1: Install from Homebrew

This is the simplest way to install *gitlab-mcp* when you're on Mac.

```sh
# Add the gitlab-mcp tap repository
brew tap fforster/gitlab-mcp https://gitlab.com/fforster/homebrew-gitlab-mcp.git

# Install gitlab-mcp
brew install gitlab-mcp
```

### Option 2: Download Precompiled Binaries

The [GitLab Releases](https://gitlab.com/fforster/gitlab-mcp/-/releases) page contains pre-built binaries for Linux, Mac OS, and Windows, for x86_64 and ARM64 architectures.

Download the appropriate `.tar.gz` file, extract it, and you will find the binary in the extracted directory.
The binary does not depend on any other files.

### Option 3: Build from source

This is recommended if you plan on contributing features or bug fixes to *gitlab-mcp*.

1. Clone this repository
2. Build the binary with `go build`, this creates `gitlab-mcp` in the current directory.
   Use `-o <path>` to chose a different filename (or directory).

### Option 4: Docker

A pre-built Docker image is available:

```bash
# Use with gitlab.com (default)
docker run -i -e GITLAB_TOKEN=your_gitlab_token registry.gitlab.com/fforster/gitlab-mcp:latest

# Use with custom GitLab instance
docker run -i -e GITLAB_TOKEN=your_gitlab_token -e GITLAB_URL=https://gitlab.example.com registry.gitlab.com/fforster/gitlab-mcp:latest
```

## Authentication

_gitlab-mcp_ offers two authentication methods:

- OAuth2 Token (preferred)
- Personal Access Token

### OAuth2 Token

Run `gitlab-mcp auth` to generate an OAuth2 token.
This is the preferred method because it expires in a few days, reducing the risk of your token being exposed or misused.

```bash
# Run the OAuth flow to obtain tokens
$ gitlab-mcp auth
Please visit the following URL to authorize the application:
https://gitlab.com/oauth/device?user_code=WDJB-MJHT

Waiting to retrieve an access token...
Success.
```

The command prints an authentication URL.
Follow the link and authorize the application to complete the flow.
_gitlab-mcp_ will automatically detect when you have completed the OAuth flow and will then store the access token in a file.

For custom GitLab instances, use the `--base-url` flag to specify the URL of the GitLab instance.
For example:

```bash
gitlab-mcp auth --base-url https://gitlab.example.com
```

### Personal Access Token

Alternatively, _gitlab-mcp_ can use a personal access token for authentication.
Generate one from your [GitLab User Settings](https://gitlab.com/-/user_settings/personal_access_tokens) with the `api` scope, or `read_api` for read-only access.

Once created, ensure the token is available to _gitlab-mcp_ in the environment variable `GITLAB_TOKEN`.

## Usage with Claude Desktop

To use the GitLab MCP server with Claude Desktop, add the following to your Claude Desktop configuration file:

### Using the binary

```json
{
  "mcpServers": {
    "GitLab": {
      "command": "/opt/homebrew/bin/gitlab-mcp",
    }
  }
}
```

### Using Docker

```json
{
  "mcpServers": {
    "GitLab": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "--pull=always",
        "-e", "GITLAB_TOKEN",
        "registry.gitlab.com/fforster/gitlab-mcp:latest",
        "--",
        "--base-url", "https://gitlab.example.com"
      ],
      "env": {
        "GITLAB_TOKEN": "your_gitlab_token"
      }
    }
  }
}
```

> [!note]
> Remove the `--base-url` argument to use the default `gitlab.com` instance.

**Explanation:**

- `-i` is required so Docker keeps `STDIN` open.
- `--rm` cleans up the container when *gitlab-mcp* exits.
- `--pull=always` ensures that Docker downloads the latest version of *gitlab-mcp* before running.
- `-e GITLAB_TOKEN` exports the `GITLAB_TOKEN` environment variable to *gitlab-mcp*.
- `--base-url` specifies a custom GitLab instance URL (optional).

### Config file location

The configuration file is typically located at:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

### Environment Variables

GitLab MCP supports the following environment variables:

- **`GITLAB_TOKEN`**: Your GitLab personal access token with `api` (or `read_api` for read-only) scope.
  Prefer to create an OAuth token using `gitlab-mcp auth` instead, which does not require any configuration via environment variables.
- **`GITLAB_URL`**: Custom GitLab instance URL (defaults to `https://gitlab.com/api/v4`).
  - Must use HTTPS scheme for security
  - Example: `https://gitlab.example.com`
  - Will be overridden by the `--base-url` flag if both are specified

You can provide these environment variables in one of two ways:

1. **Directly in the configuration**, by adding an `env` section to the configuration:

   ```json
   {
     "mcpServers": {
       "GitLab": {
         "command": "/opt/homebrew/bin/gitlab-mcp",
         "env": {
           "GITLAB_TOKEN": "your_gitlab_token",
           "GITLAB_URL": "https://gitlab.example.com"
         }
       }
     }
   }
   ```

2. **Via shell initialization files**, where your `.bashrc` or similar file sets the environment variables:

   ```json
   {
     "mcpServers": {
       "GitLab": {
         "command": "/bin/bash",
         "args": [
           "-c",
           "/opt/homebrew/bin/gitlab-mcp"
         ]
       }
     }
   }
   ```

## Supported Tools

| Tool Name | Description |
|-----------|-------------|
| `complete_all_todo_items` | Marks all pending todo items for the current user as done |
| `complete_todo_item` | Marks a single pending todo item as done |
| `create_epic` | Creates a new GitLab epic |
| `create_issue` | Creates a new GitLab issue |
| `create_snippet` | Create a new snippet |
| `delete_snippet` | Delete a snippet |
| `discussion_add_note` | Adds a new note (i.e. a reply) to an existing discussion thread |
| `discussion_delete_note` | Deletes a note from a discussion thread |
| `discussion_list` | Lists all discussions for a GitLab resource |
| `discussion_modify_note` | Modifies an existing note in a discussion thread |
| `discussion_new` | Creates a new discussion thread on a GitLab resource |
| `discussion_resolve` | Resolves or unresolves a discussion thread in a merge request |
| `download_job_artifacts_file` | Download a single artifact file from a job |
| `download_job_log` | Download a log file for a specific job |
| `edit_epic` | Updates an existing GitLab epic |
| `edit_issue` | Updates an existing GitLab issue. You can modify the issue's title, description, and metadata, and close/reopen issues. |
| `get_epic_links` | GetEpicLinks gets all child epics of an epic |
| `get_epic` | Fetches information about an epic by ID |
| `get_issue` | Get a single project issue |
| `get_issues_closed_on_merge` | Get all the issues that would be closed by merging the provided merge request |
| `get_job` | Get a single job of a project |
| `get_merge_request_approvals` | Get approvals for a merge request |
| `get_merge_request_changes` | Get information about the merge request including its files and changes |
| `get_merge_request_commits` | Get all commits associated with a merge request |
| `get_merge_request_dependencies` | Get merge request dependencies |
| `get_merge_request_participants` | Get a list of merge request participants |
| `get_merge_request_reviewers` | Get a list of merge request reviewers |
| `get_merge_request` | Get a single merge request |
| `get_repository_file_contents` | Get the contents of a single file from the repository |
| `get_snippet_content` | Get the raw content of a snippet |
| `get_snippet` | Returns the metadata of a snippet, such as title and description. File content is not returned |
| `get_user_status` | Get a user's status |
| `get_user` | Get information about a specific user or the current user. In particular, this tool can be used to resolve a username to an ID. |
| `issue_time_tracking` | Updates time tracking information for an issue, such as the time estimate and spent time. |
| `list_all_snippets` | List all snippets the user has access to |
| `list_draft_notes` | Returns a list of draft notes for the merge request |
| `list_downstream_pipelines` | Get a list of downstream pipeline triggers for a pipeline |
| `list_epic_issues` | Returns a list of issues assigned to the provided epic |
| `list_group_epics` | Get all epics for a specific group |
| `list_group_issues` | Get a list of a group's issues |
| `list_group_merge_requests` | Get all merge requests for this group |
| `list_issues` | Lists GitLab issues |
| `list_merge_request_diffs` | Get a list of merge request diff versions |
| `list_merge_request_pipelines` | Get a list of merge request pipelines |
| `list_merge_requests_related_to_issue` | Get all merge requests that are related to the specified issue |
| `list_pipeline_jobs` | Get a list of jobs for a pipeline |
| `list_project_issues` | Get a list of a project's issues |
| `list_project_merge_requests` | Get all merge requests for this project |
| `list_repository_directory` | Get a list of repository files and directories in a project |
| `list_user_events`  | Get events for users between |
| `list_user_merge_requests` | Get all merge requests authored by or assigned to a user for review |
| `list_user_snippets` | List snippets owned by the current user |
| `list_user_todos` | Get all todos for the current user, with optional filtering by state (pending, done) and a limit on the number returned (default 100). |
| `retry_job` | Retry a single job of a project |
| `set_user_status` | Set the current user's status |
| `trigger_manual_job` | Trigger a manual job for a project |
| `update_snippet` | Update an existing snippet |

## Example Usage

Once configured, you can ask Claude to interact with your GitLab instance:

- "Prioritize the open issues assigned to me."
- "List the merge requests I need to review."
- "Add a comment to issue #42 suggesting a possible solution."
- "Summarize the discussion of issue #123 in the `my-project` project."
- "Create a new snippet with the code I just shared."
- "What epics are currently open in the `my-group` group?"
- "Please explain why the pipeline of merge request !456 failed."
- "How do I use the MCP server according to the `README.md` file in the `fforster/gitlab-mcp` GitLab project."
- "On GitLab, what have reprazent and ffoster been working on between 2025-04-28
  and 2025-05-02?"
- "List all of my pending todos."
- "List the latest 100 completed todos."

## Streaming HTTP support

> [!warning]
> Streaming HTTP support in _gitlab-mcp_ is still experimental and not ready for production use.
>
> Limitations:
>
> - Only personal access tokens are supported for authentication, which can be stolen and misused by the server operator

_gitlab-mcp_ has experimental support for the streaming HTTP transport.
This allows you to use the GitLab MCP server with other applications that support the MCP protocol over HTTP.

Using the MCP server via HTTP decouples the client and server processes, allowing you to restart or upgrade the server without affecting the client.
Eventually _gitlab-mcp_ may support multi-tenant installations, enabling multiple users to connect to a single server instance.
For the time being, this is very strongly discouraged because we only support personal access tokens as an authentication mechanism, which can easily be stolen and misused by the server operator.

### Start the server

To start _gitlab-mcp_ in Streaming HTTP mode, you need to specify the `--transport http` flag.
The server listens on the address specified by the `--listen-addr` flag (default `:8080`).

Use the `--tls-cert` and `--tls-key` flags to enable TLS support.

```sh
$ ./gitlab-mcp --transport http --listen-addr localhost:3001
2025/07/04 23:34:04 Listening on localhost:3001 (without TLS)
```

### Connect to the server

Please follow your IDE's or application's documentation to configure the MCP server with streaming HTTP transport.

The MCP server listens to the `/mcp` endpoint. With the above example, you would configure the IDE or application to connect to `http://localhost:3001/mcp`.

> [!caution]
> Only send Personal Access Tokens to the _gitlab-mcp_ server if you trust it completely (i.e. you're running it yourself).
> The server operator can easily steal your access token and act as you on GitLab.

> [!caution]
> Do not send Personal Access Tokens over unencrypted network connections.
> Ensure to use an HTTPS URL, or use the local loopback address, e.g. `localhost`, `::1`, or `127.0.0.1`.

_gitlab-mcp_ requires the `Private-Token` HTTP header to be set to your GitLab personal access token for authentication.
If the header is missing, a `401 Unauthorized` response is returned.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
