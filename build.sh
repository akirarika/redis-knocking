#!/bin/bash

VERSION="1.0.0"
BASE_DIR="packages"
mkdir -p "$BASE_DIR"

MODULE_NAME=$(head -n 1 go.mod | awk '{print $2}')

PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/386"
    "linux/amd64"
    "linux/arm"
    "linux/arm64"
    "windows/386"
    "windows/amd64"
    "windows/arm64"
    "freebsd/amd64"
    "freebsd/arm64"
)

# Function to build for a platform
build_for_platform() {
    local GOOS=$1
    local GOARCH=$2
    
    # Create platform-specific directory structure
    local PLATFORM_NAME="redis-knocking-${GOOS}-${GOARCH}"
    local PLAT_DIR="${BASE_DIR}/${PLATFORM_NAME}"
    local BIN_DIR="${PLAT_DIR}/bin"
    mkdir -p "$BIN_DIR"

    # Determine executable name
    local EXECUTABLE_NAME="redis-knocking"
    if [ "$GOOS" = "windows" ]; then
        EXECUTABLE_NAME="${EXECUTABLE_NAME}.exe"
    fi

    local OUTPUT_PATH="${BIN_DIR}/${EXECUTABLE_NAME}"

    # Build binary
    echo "Building for $GOOS/$GOARCH..."
    env \
        GOOS=$GOOS \
        GOARCH=$GOARCH \
        CGO_ENABLED=0 \
        go build -ldflags="-s -w" -o "$OUTPUT_PATH" index.go

    if [ $? -ne 0 ]; then
        echo "Error building for $GOOS/$GOARCH"
        return 1
    fi

    # Create package.json
    local BIN_RELATIVE_PATH="bin/${EXECUTABLE_NAME}"
    cat > "${PLAT_DIR}/package.json" <<EOF
{
    "name": "${PLATFORM_NAME}",
    "version": "${VERSION}",
    "bin": {
        "${PLATFORM_NAME}": "${BIN_RELATIVE_PATH}"
    },
    "publishConfig": {
        "access": "public"
    }
}
EOF
}

# Build for all platforms
for platform in "${PLATFORMS[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    build_for_platform "$GOOS" "$GOARCH" || exit 1
done

echo "Build complete! Packages are in ${BASE_DIR}/ directory."

# Ask about publishing
read -p "Do you want to publish all packages to npm? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Publishing packages to npm..."
    for platform in "${PLATFORMS[@]}"; do
        GOOS=${platform%/*}
        GOARCH=${platform#*/}
        PLATFORM_NAME="redis-knocking-${GOOS}-${GOARCH}"
        PLAT_DIR="${BASE_DIR}/${PLATFORM_NAME}"
        
        echo "Publishing ${PLATFORM_NAME}..."
        (cd "$PLAT_DIR" && npm publish --access public)
        
        if [ $? -ne 0 ]; then
            echo "Error publishing ${PLATFORM_NAME}"
            exit 1
        fi
    done
    echo "All packages published successfully!"
else
    echo "Skipping npm publish."
fi