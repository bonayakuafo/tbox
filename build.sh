#!/usr/bin/env bash
# tbox build script.
#
# Usage:
#   ./build.sh                       build the current platform, output ./dist/tbox
#   ./build.sh all                   cross-compile all preset platforms into ./dist/
#   ./build.sh <goos> <goarch>       build a specific platform
#
# The version is taken from git (nearest tag, or the short SHA) and embedded
# into the binary via -ldflags "-X main.version=<version>". Override by
# exporting VERSION=<value> before running.

set -euo pipefail

NAME="tbox"
ENTRY="tbox.go"
OUT_DIR="dist"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X main.version=${VERSION}"

PLATFORMS=(
	"linux/amd64" "linux/arm64"
	"darwin/amd64" "darwin/arm64"
	"windows/amd64" "windows/arm64"
)

# Build a single platform. Output: dist/<name>-<goos>-<goarch>[.exe]
build_one() {
	local goos="$1" goarch="$2"
	local ext=""
	[ "$goos" = "windows" ] && ext=".exe"
	local out="${OUT_DIR}/${NAME}-${goos}-${goarch}${ext}"
	mkdir -p "$OUT_DIR"
	echo "Building ${goos}/${goarch} -> ${out}"
	CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
		go build -trimpath -ldflags "$LDFLAGS" -o "$out" "$ENTRY"
}

# Build all preset platforms. A single failure is recorded but does not abort.
build_all() {
	local failed=()
	for p in "${PLATFORMS[@]}"; do
		if ! build_one "${p%%/*}" "${p##*/}"; then
			echo "  warning: ${p} build failed, skipped" >&2
			failed+=("$p")
		fi
	done
	if [ "${#failed[@]}" -gt 0 ]; then
		echo "Done. Failures: ${failed[*]}"
		return 1
	fi
	echo "All done. Artifacts in ${OUT_DIR}/"
}

# Build for the current platform to ./dist/tbox[.exe].
build_local() {
	local ext=""
	[ "$(go env GOOS)" = "windows" ] && ext=".exe"
	local out="${OUT_DIR}/${NAME}${ext}"
	mkdir -p "$OUT_DIR"
	echo "Building current platform ($(go env GOOS)/$(go env GOARCH)) -> ${out}"
	go build -trimpath -ldflags "$LDFLAGS" -o "$out" "$ENTRY"
	echo "Done: ${out} (version: ${VERSION})"
}

main() {
	case "${1:-}" in
	"") build_local ;;
	-d | all) build_all ;;
	*)
		if [ "$#" -eq 2 ]; then
			build_one "$1" "$2"
		else
			echo "Usage: $0 [ all | <goos> <goarch> ]" >&2
			exit 1
		fi
		;;
	esac
}

main "$@"
