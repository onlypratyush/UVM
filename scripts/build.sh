#!/usr/bin/env bash
# ==============================================================================
# Cross-Platform Build & Packaging Script for uvm
# ==============================================================================

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${REPO_ROOT}/dist"
VERSION="${1:-0.0.1}"

# Target platforms and architectures: OS/ARCH/EXT
TARGETS=(
    "darwin/arm64/"
    "darwin/amd64/"
    "linux/amd64/"
    "linux/arm64/"
    "windows/amd64/.exe"
    "windows/arm64/.exe"
)

echo "==> Building uvm v${VERSION}..."
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

LDFLAGS="-s -w -X main.version=${VERSION}"

for target in "${TARGETS[@]}"; do
    IFS="/" read -r OS ARCH EXT <<< "${target}"
    OUTPUT_NAME="uvm-${OS}-${ARCH}${EXT}"
    OUTPUT_PATH="${DIST_DIR}/${OUTPUT_NAME}"

    echo "  -> Compiling for ${OS}/${ARCH}..."
    GOOS="${OS}" GOARCH="${ARCH}" CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o "${OUTPUT_PATH}" "${REPO_ROOT}/main.go"

    # Create release archives
    ARCHIVE_DIR="${DIST_DIR}/uvm_v${VERSION}_${OS}_${ARCH}"
    mkdir -p "${ARCHIVE_DIR}"
    cp "${REPO_ROOT}/README.md" "${ARCHIVE_DIR}/"
    cp "${REPO_ROOT}/LICENSE" "${ARCHIVE_DIR}/"

    if [ "${OS}" = "windows" ]; then
        cp "${OUTPUT_PATH}" "${ARCHIVE_DIR}/uvm.exe"
        (cd "${DIST_DIR}" && zip -q -r "uvm_v${VERSION}_${OS}_${ARCH}.zip" "uvm_v${VERSION}_${OS}_${ARCH}")
    else
        cp "${OUTPUT_PATH}" "${ARCHIVE_DIR}/uvm"
        (cd "${DIST_DIR}" && tar -czf "uvm_v${VERSION}_${OS}_${ARCH}.tar.gz" "uvm_v${VERSION}_${OS}_${ARCH}")
    fi

    rm -rf "${ARCHIVE_DIR}"
done

echo "==> Generating SHA256 Checksums..."
(
    cd "${DIST_DIR}"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum uvm* > checksums.txt
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 uvm* > checksums.txt
    fi
)

echo "==> Build complete! Output files in ${DIST_DIR}:"
ls -la "${DIST_DIR}"
