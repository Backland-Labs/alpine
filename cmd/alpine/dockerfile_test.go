package main

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
		markers   []string
	}{
		{
			name:      "ubuntu 24.04",
			baseImage: "ubuntu:24.04",
			markers: []string{
				"FROM ubuntu:24.04",
				"apt-get update",
				"apt-get install -y --no-install-recommends",
				"git",
				"githubcli-archive-keyring.gpg",
				"github-cli.list",
				"install -y --no-install-recommends gh",
				"useradd -m -s /bin/bash claude",
				"ssh-keyscan",
				"credential.helper",
				"USER claude",
				"WORKDIR /workspace",
				"claude.ai/install.sh",
			},
		},
		{
			name:      "different base image",
			baseImage: "node:20-bookworm",
			markers: []string{
				"FROM node:20-bookworm",
				"apt-get",
				"useradd",
				"git",
				"credential.helper",
				"USER claude",
				"WORKDIR /workspace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := string(generateDockerfile(tt.baseImage))
			for _, m := range tt.markers {
				if !strings.Contains(df, m) {
					t.Errorf("Dockerfile missing %q", m)
				}
			}
		})
	}
}

func TestDockerfileHash(t *testing.T) {
	df1 := generateDockerfile("ubuntu:24.04")
	df2 := generateDockerfile("ubuntu:22.04")

	h1 := dockerfileHash(df1)
	h2 := dockerfileHash(df2)

	// Length must be 16 hex chars.
	if len(h1) != 16 {
		t.Fatalf("hash length = %d, want 16", len(h1))
	}

	// Deterministic: same input -> same hash.
	if dockerfileHash(df1) != h1 {
		t.Fatal("hash is not deterministic")
	}

	// Different input -> different hash.
	if h1 == h2 {
		t.Fatal("different inputs produced same hash")
	}
}
