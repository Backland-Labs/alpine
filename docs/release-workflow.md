# Alpine Release Workflow

This document describes the Git-flow based release process for Alpine, managing releases from the `develop` branch to the `main` branch.

## Branch Strategy

- **`develop`** - Active development branch where features are merged
- **`main`** - Production-ready code, only updated via releases
- **`feature/*`** - Feature branches created from `develop`
- **`release/*`** - Release preparation branches

## Release Process

### 1. Start a Release

When ready to create a new release from `develop`:

```bash
# Ensure develop is up to date
git checkout develop
git pull origin develop

# Create release branch
git checkout -b release/v0.6.0

# Update version and changelog
# - Update version in any relevant files
# - Update CHANGELOG.md (change [Unreleased] to [0.6.0] - YYYY-MM-DD)
# - Add new [Unreleased] section at top

git add CHANGELOG.md
git commit -m "chore: Prepare release v0.6.0"
```

### 2. Finalize the Release

```bash
# Merge to main
git checkout main
git pull origin main
git merge --no-ff release/v0.6.0

# Tag the release
git tag -a v0.6.0 -m "Release v0.6.0"

# Merge back to develop
git checkout develop
git merge --no-ff release/v0.6.0

# Push everything
git push origin main
git push origin develop
git push origin v0.6.0

# Delete release branch
git branch -d release/v0.6.0
```

### 3. Quick Release Script

For convenience, use the release script:

```bash
./scripts/create-release.sh 0.6.0
```

This script automates the entire process.

## Hotfix Process

For urgent fixes to production:

```bash
# Create hotfix from main
git checkout main
git checkout -b hotfix/v0.6.1

# Make fixes and update version/changelog
# ...

# Merge to main and develop
git checkout main
git merge --no-ff hotfix/v0.6.1
git tag -a v0.6.1 -m "Hotfix v0.6.1"

git checkout develop
git merge --no-ff hotfix/v0.6.1

# Push and cleanup
git push origin main
git push origin develop
git push origin v0.6.1
git branch -d hotfix/v0.6.1
```

## Release Checklist

Before starting a release:

- [ ] All planned features merged to `develop`
- [ ] All tests passing on `develop` (`make test-all`)
- [ ] Code formatted (`make fmt`)
- [ ] Linter passing (`make lint`)
- [ ] CHANGELOG.md updated with all changes

## Version Numbering

Follow Semantic Versioning (MAJOR.MINOR.PATCH):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes

## First-Time Setup

If you haven't created a `develop` branch yet:

```bash
git checkout main
git checkout -b develop
git push -u origin develop
```

Set `develop` as the default branch in GitHub repository settings for PRs.