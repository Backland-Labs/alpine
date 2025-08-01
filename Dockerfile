# Build stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o alpine cmd/alpine/main.go

# Runtime stage
FROM debian:bookworm

# Install runtime dependencies and development tools
RUN apt-get update && apt-get install -y \
    # Version control
    git \
    # Basic utilities
    ca-certificates \
    curl \
    wget \
    gnupg \
    lsb-release \
    build-essential \
    # Node.js and npm
    && curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get install -y nodejs \
    # Python 3
    python3 \
    python3-venv \
    # Ruby and gem
    ruby-full \
    # GitHub CLI
    && type -p curl >/dev/null || (apt-get update && apt-get install curl -y) \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh \
    # Clean up
    && rm -rf /var/lib/apt/lists/*

# Install Rust and cargo
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install UV (Python package manager)
RUN curl -LsSf https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"

# Install Claude Code and Gemini CLI via npm
RUN npm install -g @anthropic-ai/claude-code @google/gemini-cli

# Create non-root user
RUN useradd -m -u 1000 alpine

# Copy binary from builder
COPY --from=builder /app/alpine /usr/local/bin/alpine

# Set ownership
RUN chown -R alpine:alpine /usr/local/bin/alpine

# Copy Rust and UV installations to user directory
RUN cp -r /root/.cargo /home/alpine/.cargo && \
    cp -r /root/.rustup /home/alpine/.rustup && \
    cp -r /root/.local /home/alpine/.local && \
    chown -R alpine:alpine /home/alpine/.cargo /home/alpine/.rustup /home/alpine/.local

# Switch to non-root user
USER alpine

# Update PATH for the alpine user
ENV PATH="/home/alpine/.cargo/bin:/home/alpine/.local/bin:${PATH}"

# Set working directory
WORKDIR /workspace

# Accept API keys and tokens as environment variables
ENV GEMINI_API_KEY=""
ENV CLAUDE_CODE_OAUTH_TOKEN=""

# Expose port for OAuth callback (Claude Code default)
EXPOSE 5173
EXPOSE 3001

# Entry point
ENTRYPOINT ["alpine", "--serve"]
