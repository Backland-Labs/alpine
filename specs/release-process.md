# Release Process Specification

This document outlines the process for creating and publishing a new release of the Alpine CLI. Following these steps ensures that releases are consistent, well-documented, and stable.

## 1. Versioning

This project adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

- **MAJOR** version for incompatible API changes.
- **MINOR** version for adding functionality in a backward-compatible manner.
- **PATCH** version for backward-compatible bug fixes.

## 2. Pre-Release Checklist

Before creating a new release, ensure the following steps are completed on the `main` branch:

1.  **Ensure `main` is up-to-date:**
    ```bash
    git checkout main
    git pull origin main
    ```

2.  **Run all checks and tests:**
    ```bash
    make fmt
    make lint
    make test-all
    ```
    All checks must pass. Do not proceed if there are any failures.

3.  **Update the Changelog:**
    - Open `CHANGELOG.md`.
    - Change the `[Unreleased]` section to the new version number (e.g., `[0.6.0] - YYYY-MM-DD`).
    - Add a new `[Unreleased]` section at the top for future changes.
    - Commit the changelog update:
      ```bash
      git add CHANGELOG.md
      git commit -m "docs: Update changelog for v0.6.0"
      ```

## 3. Creating the Release

Once the pre-release checklist is complete, create the release tag and push it to the repository.

1.  **Tag the release:**
    Create an annotated Git tag for the new version.
    ```bash
    git tag -a v0.6.0 -m "Release v0.6.0"
    ```

2.  **Push the changes and tag:**
    ```bash
    git push origin main
    git push origin v0.6.0
    ```

## 4. GitHub Release

After pushing the tag, a GitHub Action workflow will typically trigger to create the release and upload the artifacts. If manual creation is needed:

1.  Go to the "Releases" page in the GitHub repository.
2.  Click "Draft a new release".
3.  Select the tag you just pushed (e.g., `v0.6.0`).
4.  Set the release title to the version (e.g., `v0.6.0`).
5.  Copy the relevant section from `CHANGELOG.md` into the release description.
6.  Upload the binary artifacts from the `release/` directory.
7.  Publish the release.
