# How to use gitlab-mcp with self-managed GitLab instances

When using gitlab-mcp with self-managed GitLab instances, you need to create a client ID for OAuth authentication.
This section explains how to set up authentication for your self-managed GitLab instance.

## Authenticating with OAuth

### Creating an OAuth Application for Self-managed GitLab

1. Log in to your self-managed GitLab instance
2. Navigate to **User Settings** by clicking on your avatar in the top-right corner → **Preferences**
3. Select **Applications** from the left sidebar
4. Click **Add new application** to create a new OAuth application with the following settings:
   * **Name**: `gitlab-mcp` (or any name of your choosing)
   * **Redirect URI**: `urn:ietf:wg:oauth:2.0:oob`
   * **Confidential**: Uncheck this box (set to `false`)
   * **Scopes**: Select `api` (or `read_api` for read-only access)
5. Click **Save application**
6. GitLab will generate an **Application ID** (aka. Client ID). Make note of the **Application ID** as you'll need it in the next steps.

### Authenticating with your Self-managed GitLab Instance

Once you have created the OAuth application, you can authenticate using the following steps:

1. Run the `gitlab-mcp auth` command with your instance URL and client ID:

   ```bash
   gitlab-mcp auth --base-url "https://gitlab.example.com" --client-id "YOUR_CLIENT_ID"
   ```

2. Follow the URL displayed in the terminal to authorize the application
3. Complete the authentication flow in your browser
4. The command will store the access token for future use

### Configuring Claude Desktop for Self-managed GitLab

In your Claude Desktop configuration file, include both the `--base-url` and `--client-id` parameters:

```json
{
  "mcpServers": {
    "GitLab": {
      "command": "/opt/homebrew/bin/gitlab-mcp",
      "args": [
        "--base-url", "https://gitlab.example.com",
        "--client-id", "YOUR_CLIENT_ID"
      ]
    }
  }
}
```

### Using Docker with Self-managed GitLab

When using Docker, include both the base URL and client ID in your configuration:

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
        "--base-url", "https://gitlab.example.com",
        "--client-id", "YOUR_CLIENT_ID"
      ],
      "env": {
        "GITLAB_TOKEN": "your_gitlab_token"
      }
    }
  }
}
```

## Authenticating with a Personal Access Token

While OAuth authentication is preferred, you can also use a _Personal Access Token_ (PAT) with your self-managed GitLab instance:

> [!warning]
> OAuth authentication is recommended over personal access tokens as OAuth tokens expire after a short period, reducing security risks.

1. Generate a Personal Access Token from your GitLab instance's **User Settings** → **Access Tokens** with the `api` scope
2. Configure gitlab-mcp to use your self-managed instance by setting the `GITLAB_TOKEN` environment variable:

   ```bash
   export GITLAB_TOKEN="your_personal_access_token"
   gitlab-mcp --base-url "https://gitlab.example.com"
   ```

   Or in your Claude Desktop configuration:

   ```json
   {
     "mcpServers": {
       "GitLab": {
         "command": "/opt/homebrew/bin/gitlab-mcp",
         "args": [
           "--base-url", "https://gitlab.example.com"
         ],
         "env": {
           "GITLAB_TOKEN": "your_personal_access_token"
         }
       }
     }
   }
   ```
