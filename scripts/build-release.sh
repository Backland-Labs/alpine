#!/bin/bash
set -e

# Build release binaries for River v0.2.0

VERSION="v0.2.0"
BINARY_NAME="river"

echo "Building River ${VERSION} release binaries..."

# Create release directory
mkdir -p release

# Build for each platform
platforms=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r -a platform_parts <<< "$platform"
    GOOS="${platform_parts[0]}"
    GOARCH="${platform_parts[1]}"
    
    output_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        output_name+=".exe"
    fi
    
    echo "Building for ${GOOS}/${GOARCH}..."
    
    # Build with version information
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "release/${output_name}" \
        cmd/river/main.go
    
    # Create archive
    if [ "$GOOS" = "windows" ]; then
        # Create zip for Windows
        (cd release && zip "${output_name}.zip" "${output_name}")
    else
        # Create tar.gz for Unix systems
        (cd release && tar -czf "${output_name}.tar.gz" "${output_name}")
    fi
    
    # Remove the uncompressed binary
    rm "release/${output_name}"
done

echo "Release binaries created in ./release/"
ls -la release/