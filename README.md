# GitLab MCP

A Model Context Protocol (MCP) server for GitLab, enabling Claude to interact directly with your GitLab instance.

## Overview

GitLab MCP implements the [Model Context Protocol](https://github.com/anthropics/model-context-protocol) for GitLab, allowing AI assistants like Claude to access and manipulate GitLab resources. This integration enables Claude to work with:

- Epics
- Issues
- Jobs
- Merge Requests
- Snippets
- User information

## Installation

### Option 1: Build from source

1. Clone this repository
2. Build the binary with `go build`
3. Set up your GitLab token as an environment variable:
   ```bash
   export GITLAB_TOKEN=your_gitlab_token
   ```

### Option 2: Docker

A pre-built Docker image is available:

```bash
docker run -e GITLAB_TOKEN=your_gitlab_token registry.gitlab.com/fforster/gitlab-mcp:latest
```

The Docker image contains the compiled GitLab MCP server and is the simplest way to get started.

## Usage with Claude Desktop

To use the GitLab MCP server with Claude Desktop, add the following to your Claude Desktop configuration file:

### Using the binary

```json
{
  "mcpServers": {
    "GitLab": {
      "command": "/path/to/gitlab-mcp",
      "env": {
        "GITLAB_TOKEN": "your_gitlab_token"
      }
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
        "-e", "GITLAB_TOKEN",
        "registry.gitlab.com/fforster/gitlab-mcp:latest"
      ],
      "env": {
        "GITLAB_TOKEN": "your_gitlab_token"
      }
    }
  }
}
```

### Config file location

The configuration file is typically located at:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

### Environment Variables

You can provide the GitLab token in one of two ways:

1. **Directly in the configuration**, by adding an `env` section. This is what the above examples use.
2. **Via shell initialization files**, where your `.bashrc` or similar file sets the `GITLAB_TOKEN` environment variable:

   ```json
   {
     "mcpServers": {
       "GitLab": {
         "command": "/bin/bash",
         "args": [
           "-c",
           "/path/to/gitlab-mcp"
         ],
       }
     }
   }
   ```

## Supported Tools

| Tool Name | Description |
|-----------|-------------|
| `get_user` | Get information about a specific user or the current user. In particular, this tool can be used to resolve a username to an ID. |
| `get_user_status` | Get a user's status |
| `set_user_status` | Set the current user's status |
| `list_group_epics` | Get all epics for a specific group |
| `get_epic` | Fetches information about an epic by ID |
| `get_epic_links` | GetEpicLinks gets all child epics of an epic |
| `list_epic_issues` | Returns a list of issues assigned to the provided epic |
| `list_user_issues` | Lists all issues assigned to a user |
| `list_group_issues` | Get a list of a group's issues |
| `list_project_issues` | Get a list of a project's issues |
| `get_issue` | Get a single project issue |
| `list_merge_requests_related_to_issue` | Get all merge requests that are related to the specified issue |
| `list_user_merge_requests` | Get all merge requests authored by or assigned to a user for review |
| `list_project_merge_requests` | Get all merge requests for this project |
| `list_group_merge_requests` | Get all merge requests for this group |
| `get_merge_request` | Get a single merge request |
| `get_merge_request_approvals` | Get approvals for a merge request |
| `get_merge_request_commits` | Get all commits associated with a merge request |
| `get_merge_request_changes` | Get information about the merge request including its files and changes |
| `list_merge_request_diffs` | Get a list of merge request diff versions |
| `get_merge_request_participants` | Get a list of merge request participants |
| `get_merge_request_reviewers` | Get a list of merge request reviewers |
| `list_merge_request_pipelines` | Get a list of merge request pipelines |
| `get_issues_closed_on_merge` | Get all the issues that would be closed by merging the provided merge request |
| `get_merge_request_dependencies` | Get merge request dependencies |
| `list_user_snippets` | List snippets owned by the current user |
| `list_all_snippets` | List all snippets the user has access to |
| `get_snippet` | Returns the metadata of a snippet, such as title and description. File content is not returned |
| `get_snippet_content` | Get the raw content of a snippet |
| `create_snippet` | Create a new snippet |
| `update_snippet` | Update an existing snippet |
| `delete_snippet` | Delete a snippet |
| `list_pipeline_jobs` | Get a list of jobs for a pipeline |
| `list_downstream_pipelines` | Get a list of downstream pipeline triggers for a pipeline |
| `get_job` | Get a single job of a project |
| `download_job_artifacts_file` | Download a single artifact file from a job |
| `download_job_log` | Download a log file for a specific job |
| `retry_job` | Retry a single job of a project |
| `trigger_manual_job` | Trigger a manual job for a project |

## Example Usage

Once configured, you can ask Claude to interact with your GitLab instance:

- "Show me open issues assigned to me"
- "List the merge requests I need to review"
- "Show me the details of issue #123 in the my-project project"
- "Create a new snippet with the code I just shared"
- "What epics are currently open in the my-group group?"

## License

This project is licensed under the MIT License - see the LICENSE file for details.
