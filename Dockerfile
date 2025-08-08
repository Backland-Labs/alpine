# Build stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies and build in one layer
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o alpine cmd/alpine/main.go

# Runtime stage - use debian:bookworm-slim for smaller base
FROM debian:bookworm-slim

# Create user first to avoid permission issues
RUN useradd -m -u 1000 alpine

# Install all dependencies in a single layer with aggressive cleanup
RUN apt-get update && \
    # Install basic packages
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        wget \
        gnupg \
        lsb-release \
        git \
        build-essential \
        && \
    # Install Node.js
    curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    # Install Python 3
    apt-get install -y --no-install-recommends python3 python3-venv && \
    # Install Ruby
    apt-get install -y --no-install-recommends ruby-full && \
    # Install GitHub CLI
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && \
    apt-get install -y --no-install-recommends gh && \
    # Install global npm packages
    npm install -g @anthropic-ai/claude-code @google/gemini-cli && \
    # Clean npm cache
    npm cache clean --force && \
    # Clean apt caches and lists
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    # Remove unnecessary files
    rm -rf /usr/share/doc /usr/share/man /usr/share/info /usr/share/locale

# Switch to non-root user for remaining installations
USER alpine
WORKDIR /home/alpine

# Install Rust for the alpine user directly (no copying needed)
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --no-modify-path --default-toolchain stable --profile minimal && \
    # Install UV for the alpine user directly
    curl -LsSf https://astral.sh/uv/install.sh | sh && \
    # Clean up Rust installation files
    rm -rf /home/alpine/.cargo/registry/cache /home/alpine/.cargo/git/db && \
    # Clean up UV installation files
    rm -rf /home/alpine/.local/share/uv/cache

# Update PATH for the alpine user
ENV PATH="/home/alpine/.cargo/bin:/home/alpine/.local/bin:${PATH}"

# Copy binary from builder as root, then fix ownership
USER root
COPY --from=builder /app/alpine /usr/local/bin/alpine
RUN chown alpine:alpine /usr/local/bin/alpine

# Switch back to non-root user
USER alpine
WORKDIR /workspace

# Environment variables
ENV GEMINI_API_KEY=""
ENV CLAUDE_CODE_OAUTH_TOKEN=""
ENV GITHUB_TOKEN=""

# Expose ports
EXPOSE 5173 3001

# Entry point
ENTRYPOINT ["alpine", "--serve"]
