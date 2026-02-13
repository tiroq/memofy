#!/bin/bash
set -e

# Memofy Release Builder
# Builds cross-platform binaries for GitHub releases

# Print usage if --help requested (check this first)
if [[ "$1" == "--help" || "$2" == "--help" ]]; then
    echo "Usage: bash scripts/build-release.sh [VERSION] [OPTIONS]"
    echo ""
    echo "VERSION: Release version (default: 0.1.0)"
    echo ""
    echo "OPTIONS:"
    echo "  --macos-only      Build only for macOS (default)"
    echo "  --all-platforms   Build for macOS, Linux, and Windows"
    echo "  --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  bash scripts/build-release.sh 0.2.0          # Build v0.2.0 for macOS only"
    echo "  bash scripts/build-release.sh 0.2.0 --all-platforms  # Build v0.2.0 for all platforms"
    exit 0
fi

# Now assign version and platform after help check
VERSION="${1:-0.1.0}"
BUILD_ALL_PLATFORMS="${2:---macos-only}"
BUILD_DIR="build/release"
DIST_DIR="dist"

echo "=== Memofy Release Builder ==="
echo "Building version: $VERSION"
echo ""

# Validate platform option
if [[ "$BUILD_ALL_PLATFORMS" != "--macos-only" && "$BUILD_ALL_PLATFORMS" != "--all-platforms" ]]; then
    echo "❌ Invalid option: $BUILD_ALL_PLATFORMS"
    echo "Use --help for usage information"
    exit 1
fi

BUILD_MACOS=true
BUILD_LINUX=false
BUILD_WINDOWS=false

if [[ "$BUILD_ALL_PLATFORMS" == "--all-platforms" ]]; then
    BUILD_LINUX=true
    BUILD_WINDOWS=true
    echo "Building for: macOS, Linux, Windows"
else
    echo "Building for: macOS only"
fi
echo ""

# Create directories
mkdir -p "$BUILD_DIR"
mkdir -p "$DIST_DIR"

# Build for macOS (arm64 and x86_64)
build_macos() {
    local arch=$1
    local output_dir="$BUILD_DIR/memofy-$VERSION-darwin-$arch"
    
    echo "Building for macOS $arch..."
    mkdir -p "$output_dir"
    
    GOOS=darwin GOARCH=$arch CGO_ENABLED=1 go build \
        -o "$output_dir/memofy-core" \
        -ldflags "-X main.Version=$VERSION" \
        cmd/memofy-core/main.go
    
    GOOS=darwin GOARCH=$arch CGO_ENABLED=1 go build \
        -o "$output_dir/memofy-ui" \
        -ldflags "-X main.Version=$VERSION" \
        cmd/memofy-ui/main.go
    
    # Copy supporting files
    cp README.md "$output_dir/"
    cp LICENSE "$output_dir/"
    cp -r configs "$output_dir/"
    cp -r scripts "$output_dir/"
    
    # Create zip archive
    cd "$BUILD_DIR"
    zip -r "../$DIST_DIR/memofy-$VERSION-darwin-$arch.zip" "memofy-$VERSION-darwin-$arch"
    cd - > /dev/null
    
    echo "✓ Created $DIST_DIR/memofy-$VERSION-darwin-$arch.zip"
}

# Build for Linux
build_linux() {
    local arch=$1
    local output_dir="$BUILD_DIR/memofy-$VERSION-linux-$arch"
    
    echo "Building for Linux $arch..."
    mkdir -p "$output_dir"
    
    GOOS=linux GOARCH=$arch CGO_ENABLED=0 go build \
        -o "$output_dir/memofy-core" \
        -ldflags "-X main.Version=$VERSION" \
        cmd/memofy-core/main.go
    
    GOOS=linux GOARCH=$arch CGO_ENABLED=0 go build \
        -o "$output_dir/memofy-ui" \
        -ldflags "-X main.Version=$VERSION" \
        cmd/memofy-ui/main.go
    
    # Copy supporting files
    cp README.md "$output_dir/"
    cp LICENSE "$output_dir/"
    cp -r configs "$output_dir/"
    cp -r scripts "$output_dir/"
    
    # Create tar.gz archive
    cd "$BUILD_DIR"
    tar czf "../$DIST_DIR/memofy-$VERSION-linux-$arch.tar.gz" "memofy-$VERSION-linux-$arch"
    cd - > /dev/null
    
    echo "✓ Created $DIST_DIR/memofy-$VERSION-linux-$arch.tar.gz"
}

# Build for Windows
build_windows() {
    local arch=$1
    local output_dir="$BUILD_DIR/memofy-$VERSION-windows-$arch"
    
    echo "Building for Windows $arch..."
    mkdir -p "$output_dir"
    
    GOOS=windows GOARCH=$arch CGO_ENABLED=0 go build \
        -o "$output_dir/memofy-core.exe" \
        -ldflags "-X main.Version=$VERSION" \
        cmd/memofy-core/main.go
    
    GOOS=windows GOARCH=$arch CGO_ENABLED=0 go build \
        -o "$output_dir/memofy-ui.exe" \
        -ldflags "-X main.Version=$VERSION" \
        cmd/memofy-ui/main.go
    
    # Copy supporting files
    cp README.md "$output_dir/"
    cp LICENSE "$output_dir/"
    cp -r configs "$output_dir/"
    cp -r scripts "$output_dir/"
    
    # Create zip archive
    cd "$BUILD_DIR"
    zip -r "../$DIST_DIR/memofy-$VERSION-windows-$arch.zip" "memofy-$VERSION-windows-$arch"
    cd - > /dev/null
    
    echo "✓ Created $DIST_DIR/memofy-$VERSION-windows-$arch.zip"
}

# Main build process
echo "Building release artifacts..."
echo ""

# macOS builds (ARM64 for Apple Silicon, x86_64 for Intel)
build_macos "arm64"
build_macos "amd64"

# Linux builds (conditional)
if [[ "$BUILD_LINUX" == "true" ]]; then
    build_linux "amd64"
    build_linux "arm64"
fi

# Windows builds (conditional)
if [[ "$BUILD_WINDOWS" == "true" ]]; then
    build_windows "amd64"
    build_windows "arm64"
fi

echo ""
echo "=== Release Complete ==="
echo "Artifacts created in: $DIST_DIR/"
echo ""
echo "To create a GitHub release:"
echo "  1. Tag the commit: git tag v$VERSION"
echo "  2. Push tag: git push origin v$VERSION"
echo "  3. Create release on GitHub: https://github.com/tiroq/memofy/releases/new?tag=v$VERSION"
echo "  4. Upload artifacts from $DIST_DIR/"
echo ""
echo "Files ready for upload:"
ls -lh "$DIST_DIR/" | grep -v "^total"

