# GitHub Actions Workflows

This directory contains the CI/CD pipelines for the River project.

## Workflows

### CI Workflow (`ci.yml`)

Runs on every push to main/develop branches and on pull requests.

**Jobs:**
- **Test**: Runs unit and integration tests with coverage reporting
- **Lint**: Runs golangci-lint and checks code formatting
- **Build**: Cross-compiles binaries for multiple platforms (Linux, macOS, Windows)

**Required Secrets:**
- `LINEAR_API_KEY`: API key for Linear integration tests
- `CLAUDE_API_KEY`: API key for Claude integration tests (optional)

### Release Workflow (`release.yml`)

Triggers when a new tag is pushed (e.g., `v1.0.0`).

**Jobs:**
- **Build**: Creates release binaries for all supported platforms
- **Release**: Creates a GitHub release with:
  - Automated changelog generation
  - Platform-specific installation instructions
  - Compressed binaries as release assets

## Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64 Intel, arm64 Apple Silicon)
- Windows (amd64)

## Testing Workflows Locally

Use [act](https://github.com/nektos/act) to test workflows locally:

```bash
# Test CI workflow
act -j test

# Test with secrets
act -j test -s LINEAR_API_KEY=$LINEAR_API_KEY
```

## Workflow Validation

Run the workflow validation tests:

```bash
make validate-workflows
```

This ensures the workflow files are properly formatted and contain required jobs.