# Claude Code Authentication in Docker

## Long-Lived Token Method (Recommended for CI/CD)

### Creating a Long-Lived OAuth Token
Claude Code provides a built-in command to create long-lived authentication tokens (requires Claude subscription):

```bash
# On your local machine (not in Docker)
claude setup-token
```

This command will:
1. Open a browser for OAuth authentication
2. Generate a long-lived token after successful authentication
3. Display the token for you to save

### Using the Token in Docker

1. **Environment Variable Method**:
```bash
docker run -it \
  -e CLAUDE_CODE_OAUTH_TOKEN="your-long-lived-token" \
  -e GEMINI_API_KEY="your-gemini-key" \
  alpine-image
```

2. **Docker Compose with Token**:
```yaml
version: '3.8'
services:
  alpine:
    build: .
    environment:
      - CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}
      - GEMINI_API_KEY=${GEMINI_API_KEY}
    volumes:
      - ./workspace:/workspace
```

3. **For GitHub Actions**:
Add the token as a repository secret named `CLAUDE_CODE_OAUTH_TOKEN` and use it in your workflow:
```yaml
- name: Run Claude Code
  uses: anthropics/claude-code-action@beta
  with:
    claude_code_oauth_token: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}
```

## Interactive OAuth Authentication Methods

### Method 1: Interactive Container with Port Forwarding
Run the container with port forwarding and interactive mode:

```bash
docker run -it -p 5173:5173 -e GEMINI_API_KEY="your-key" alpine-image bash
```

Then inside the container:
```bash
claude login
```

This will provide a URL to open in your browser. The OAuth callback will be handled on port 5173.

### Method 2: Using Docker Desktop with GUI Support
If using Docker Desktop on macOS/Windows, you can enable GUI support:

```bash
docker run -it \
  -e DISPLAY=host.docker.internal:0 \
  -e GEMINI_API_KEY="your-key" \
  -p 5173:5173 \
  alpine-image bash
```

### Method 3: Using X11 Forwarding (Linux)
On Linux with X11:

```bash
docker run -it \
  -e DISPLAY=$DISPLAY \
  -v /tmp/.X11-unix:/tmp/.X11-unix:rw \
  -e GEMINI_API_KEY="your-key" \
  -p 5173:5173 \
  alpine-image bash
```

### Method 4: API Key Authentication (if supported)
Check if Claude Code supports API key authentication as an alternative:

```bash
export ANTHROPIC_API_KEY="your-claude-api-key"
```

### Method 5: Volume Mount for Persistent Auth
Mount a volume to persist authentication between container runs:

```bash
docker run -it \
  -v claude-auth:/home/alpine/.config/claude \
  -e GEMINI_API_KEY="your-key" \
  -p 5173:5173 \
  alpine-image bash
```

## Docker Compose Example

```yaml
version: '3.8'
services:
  alpine:
    build: .
    ports:
      - "5173:5173"
    environment:
      - GEMINI_API_KEY=${GEMINI_API_KEY}
    volumes:
      - claude-auth:/home/alpine/.config/claude
      - ./workspace:/workspace
    stdin_open: true
    tty: true

volumes:
  claude-auth:
```

## Troubleshooting

1. **Browser doesn't open**: Copy the authentication URL from the terminal and open it manually
2. **Port already in use**: Change the host port mapping (e.g., `-p 5174:5173`)
3. **Authentication persists**: Use the volume mount approach to save credentials