package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

// generateDockerfile produces a Dockerfile that layers Claude CLI and git
// on top of the provided base image. It creates a non-root "claude" user
// and hardcodes apt-get with --no-install-recommends.
func generateDockerfile(baseImage string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "FROM %s\n", baseImage)
	buf.WriteString(`
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl openssh-client ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash claude

# Pre-populate GitHub SSH host keys so git clone over SSH does not prompt.
RUN mkdir -p /home/claude/.ssh \
    && ssh-keyscan -t ed25519,rsa github.com >> /home/claude/.ssh/known_hosts 2>/dev/null \
    && chown -R claude:claude /home/claude/.ssh \
    && chmod 700 /home/claude/.ssh \
    && chmod 600 /home/claude/.ssh/known_hosts

ENV PATH="/home/claude/.local/bin:${PATH}"

# Configure git to use GITHUB_TOKEN for HTTPS auth when available.
USER claude
RUN git config --global credential.helper \
    '!f() { echo "username=x-access-token"; echo "password=${GITHUB_TOKEN:-${GH_TOKEN}}"; }; f'

RUN bash -c 'set -o pipefail && curl -fsSL https://claude.ai/install.sh | bash'
USER root
RUN ln -s /home/claude/.local/bin/claude /usr/local/bin/claude

WORKDIR /workspace
RUN chown claude:claude /workspace
USER claude
`)
	return buf.Bytes()
}

// dockerfileHash returns the first 16 characters of the SHA-256 hash of
// the Dockerfile content. Used as a Docker image tag so identical
// Dockerfiles share a single image.
func dockerfileHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)[:16]
}
