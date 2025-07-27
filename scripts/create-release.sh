#!/bin/bash
set -e

# Release script for Alpine
# Creates a release from develop to main following Git-flow

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Usage: ./scripts/create-release.sh <version>"
    echo "Example: ./scripts/create-release.sh 0.6.0"
    exit 1
fi

# Validate version format
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format X.Y.Z"
    exit 1
fi

echo "🚀 Creating release v$VERSION"

# Check if we have uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "❌ Error: You have uncommitted changes. Please commit or stash them first."
    exit 1
fi

# Ensure we have develop branch
if ! git show-ref --verify --quiet refs/heads/develop; then
    echo "❌ Error: No develop branch found. Please create it first:"
    echo "   git checkout -b develop"
    echo "   git push -u origin develop"
    exit 1
fi

# Update branches
echo "📥 Updating branches..."
git fetch origin
git checkout develop
git pull origin develop
git checkout main
git pull origin main

# Create release branch
echo "🌿 Creating release branch..."
git checkout develop
git checkout -b release/v$VERSION

# Update changelog
echo "📝 Updating CHANGELOG.md..."
DATE=$(date +%Y-%m-%d)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s/## \[Unreleased\]/## [$VERSION] - $DATE/" CHANGELOG.md
    sed -i '' "/^## \[$VERSION\]/i\\
## [Unreleased]\\
\\
### Added\\
\\
### Changed\\
\\
### Fixed\\
\\
" CHANGELOG.md
else
    # Linux
    sed -i "s/## \[Unreleased\]/## [$VERSION] - $DATE/" CHANGELOG.md
    sed -i "/^## \[$VERSION\]/i\\
## [Unreleased]\\
\\
### Added\\
\\
### Changed\\
\\
### Fixed\\
\\
" CHANGELOG.md
fi

# Commit version bump
git add CHANGELOG.md
git commit -m "chore: Prepare release v$VERSION"

# Run tests
echo "🧪 Running tests..."
make test-all || {
    echo "❌ Tests failed! Aborting release."
    git checkout develop
    git branch -D release/v$VERSION
    exit 1
}

# Merge to main
echo "🔀 Merging to main..."
git checkout main
git merge --no-ff release/v$VERSION -m "Merge branch 'release/v$VERSION'"

# Tag the release
echo "🏷️  Creating tag..."
git tag -a v$VERSION -m "Release v$VERSION"

# Merge back to develop
echo "🔀 Merging back to develop..."
git checkout develop
git merge --no-ff release/v$VERSION -m "Merge branch 'release/v$VERSION' into develop"

# Push everything
echo "📤 Pushing to remote..."
git push origin main
git push origin develop
git push origin v$VERSION

# Cleanup
echo "🧹 Cleaning up..."
git branch -d release/v$VERSION

echo "✅ Release v$VERSION completed successfully!"
echo ""
echo "Next steps:"
echo "1. Create a GitHub release from the tag v$VERSION"
echo "2. Add release notes from CHANGELOG.md"
echo "3. Announce the release"