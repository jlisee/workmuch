#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPOSITORY_DIR=$(cd -- "$SCRIPT_DIR/../.." &>/dev/null && pwd)
DIST_DIR="$REPOSITORY_DIR/dist/macos"
TEMPLATE_APP="$SCRIPT_DIR/WorkMuch.app"
IDENTITY=${WORKMUCH_CODESIGN_IDENTITY:-}
MOUNTED_DMG=""
TEMP_DIR=""

fail() {
	echo "error: $*" >&2
	exit 1
}

cleanup() {
	if [[ -n "$MOUNTED_DMG" ]]; then
		hdiutil detach "$MOUNTED_DMG" >/dev/null 2>&1 || true
	fi
	if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
		rm -rf -- "$TEMP_DIR"
	fi
}
trap cleanup EXIT

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

preflight() {
	[[ $(uname -s) == Darwin ]] || fail "darwin/universal releases must be built on macOS"
	[[ -n "$IDENTITY" ]] || fail "WORKMUCH_CODESIGN_IDENTITY must name a persistent Code Signing identity"
	[[ "$IDENTITY" != "-" ]] || fail "ad-hoc signing is not supported; use a persistent Code Signing identity"

	for command_name in go clang lipo codesign hdiutil plutil otool shasum xcode-select; do
		require_command "$command_name"
	done
	xcode-select -p >/dev/null 2>&1 || fail "Xcode Command Line Tools are required"
	[[ $(go env CGO_ENABLED) == 1 ]] || fail "CGO must be enabled"
	clang --version >/dev/null 2>&1 || fail "the Xcode clang toolchain is unavailable"
}

render_plist() {
	local target=$1
	local short_version=$2
	local bundle_version=$3

	sed \
		-e "s/__WORKMUCH_SHORT_VERSION__/$short_version/g" \
		-e "s/__WORKMUCH_BUNDLE_VERSION__/$bundle_version/g" \
		"$TEMPLATE_APP/Contents/Info.plist" >"$target"
}

build_architecture() {
	local architecture=$1
	local version=$2
	local output=$3

	echo "[Building darwin/$architecture...]"
	env \
		CGO_ENABLED=1 \
		GOOS=darwin \
		GOARCH="$architecture" \
		CC=clang \
		MACOSX_DEPLOYMENT_TARGET=13.0 \
		CGO_CFLAGS="-mmacosx-version-min=13.0" \
		CGO_LDFLAGS="-mmacosx-version-min=13.0" \
		go build \
		-trimpath \
		-ldflags "-X workmuch-go/internal/buildinfo.Version=$version" \
		-o "$output" \
		./cmd/workmuch-go
}

verify_plist_value() {
	local plist=$1
	local key=$2
	local expected=$3
	local actual

	actual=$(plutil -extract "$key" raw -o - "$plist")
	[[ "$actual" == "$expected" ]] || fail "$key is $actual, expected $expected"
}

verify_bundle() {
	local app_path=$1
	local short_version=$2
	local bundle_version=$3
	local executable="$app_path/Contents/MacOS/workmuch"
	local architectures
	local load_commands

	codesign --verify --deep --strict --verbose=2 "$app_path"
	codesign -d -r- "$app_path" >/dev/null
	plutil -lint "$app_path/Contents/Info.plist" >/dev/null
	verify_plist_value "$app_path/Contents/Info.plist" CFBundleIdentifier com.jlisee.workmuch
	verify_plist_value "$app_path/Contents/Info.plist" CFBundleExecutable workmuch
	verify_plist_value "$app_path/Contents/Info.plist" CFBundleIconFile WorkMuch
	verify_plist_value "$app_path/Contents/Info.plist" CFBundlePackageType APPL
	verify_plist_value "$app_path/Contents/Info.plist" CFBundleShortVersionString "$short_version"
	verify_plist_value "$app_path/Contents/Info.plist" CFBundleVersion "$bundle_version"
	verify_plist_value "$app_path/Contents/Info.plist" LSMinimumSystemVersion 13.0
	verify_plist_value "$app_path/Contents/Info.plist" LSUIElement true
	[[ -f "$app_path/Contents/Resources/WorkMuch.icns" ]] || fail "bundle icon is missing"

	architectures=$(lipo -archs "$executable")
	[[ " $architectures " == *" arm64 "* ]] || fail "signed executable is missing arm64"
	[[ " $architectures " == *" x86_64 "* ]] || fail "signed executable is missing x86_64"
	load_commands=$(otool -l "$executable")
	grep -Eq 'minos[[:space:]]+13\.0|version[[:space:]]+13\.0' <<<"$load_commands" ||
		fail "signed executable does not declare a macOS 13.0 deployment target"
}

build_release() {
	local version
	local short_version
	local bundle_version
	local app_path
	local dmg_root
	local artifact

	preflight
	cd "$REPOSITORY_DIR"

	echo "[Running tests...]"
	./test.sh
	echo "[Running lint...]"
	./lint.sh

	version=$(go run ./cmd/workmuch-version --format version)
	if [[ -n $(git status --porcelain=v1) ]]; then
		version="${version}.dirty"
	fi
	short_version=$(go run ./cmd/workmuch-version --format plist-short)
	bundle_version=$(go run ./cmd/workmuch-version --format plist-build)

	TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/workmuch-macos-release.XXXXXX")
	app_path="$TEMP_DIR/WorkMuch.app"
	dmg_root="$TEMP_DIR/dmg"
	cp -R "$TEMPLATE_APP" "$app_path"
	mkdir -p "$app_path/Contents/MacOS" "$app_path/Contents/Resources" "$dmg_root"
	render_plist "$app_path/Contents/Info.plist" "$short_version" "$bundle_version"
	cp "$TEMPLATE_APP/Contents/Resources/WorkMuch.icns" "$app_path/Contents/Resources/WorkMuch.icns"

	build_architecture arm64 "$version" "$TEMP_DIR/workmuch-arm64"
	build_architecture amd64 "$version" "$TEMP_DIR/workmuch-amd64"
	lipo -create "$TEMP_DIR/workmuch-arm64" "$TEMP_DIR/workmuch-amd64" \
		-output "$app_path/Contents/MacOS/workmuch"
	chmod 0755 "$app_path/Contents/MacOS/workmuch"

	echo "[Signing WorkMuch.app...]"
	codesign --force --sign "$IDENTITY" --timestamp=none "$app_path"
	verify_bundle "$app_path" "$short_version" "$bundle_version"

	cp -R "$app_path" "$dmg_root/WorkMuch.app"
	ln -s /Applications "$dmg_root/Applications"
	mkdir -p "$DIST_DIR"
	artifact="$DIST_DIR/WorkMuch_${version}_universal.dmg"
	hdiutil create -volname WorkMuch -srcfolder "$dmg_root" -format UDZO -ov "$artifact"
	hdiutil verify "$artifact"

	MOUNTED_DMG="$TEMP_DIR/mount"
	mkdir -p "$MOUNTED_DMG"
	hdiutil attach -readonly -nobrowse -mountpoint "$MOUNTED_DMG" "$artifact"
	verify_bundle "$MOUNTED_DMG/WorkMuch.app" "$short_version" "$bundle_version"
	hdiutil detach "$MOUNTED_DMG"
	MOUNTED_DMG=""

	(
		cd "$DIST_DIR"
		shasum -a 256 "$(basename "$artifact")" >checksums.txt
		shasum -a 256 --check checksums.txt
	)
	echo "Built $artifact"
}

if [[ $# -ne 0 ]]; then
	echo "usage: $0" >&2
	exit 2
fi

build_release
