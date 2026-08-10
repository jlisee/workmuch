#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPOSITORY_DIR=$(cd -- "$SCRIPT_DIR/../.." &>/dev/null && pwd)
DIST_DIR="$REPOSITORY_DIR/dist/macos"
TEMPLATE_APP="$SCRIPT_DIR/WorkMuch.app"
IDENTITY=${WORKMUCH_CODESIGN_IDENTITY:-}
INSTALLED_APP=${WORKMUCH_INSTALL_APP_PATH:-/Applications/WorkMuch.app}
INSTALLED_EXECUTABLE="$INSTALLED_APP/Contents/MacOS/workmuch"
INSTALL_BUNDLE=0
INSTALL_TERM_WAIT_ATTEMPTS=${WORKMUCH_INSTALL_TERM_WAIT_ATTEMPTS:-50}
INSTALL_KILL_WAIT_ATTEMPTS=${WORKMUCH_INSTALL_KILL_WAIT_ATTEMPTS:-20}
INSTALL_POLL_INTERVAL_SECONDS=${WORKMUCH_INSTALL_POLL_INTERVAL_SECONDS:-0.1}
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

usage() {
	echo "usage: $0 [--install]" >&2
}

parse_args() {
	if [[ $# -eq 0 ]]; then
		return
	fi
	if [[ $# -eq 1 && $1 == --install ]]; then
		INSTALL_BUNDLE=1
		return
	fi
	usage
	exit 2
}

preflight() {
	[[ $(uname -s) == Darwin ]] || fail "darwin/universal releases must be built on macOS"
	[[ -n "$IDENTITY" ]] || fail "WORKMUCH_CODESIGN_IDENTITY must name a persistent Code Signing identity"
	[[ "$IDENTITY" != "-" ]] || fail "ad-hoc signing is not supported; use a persistent Code Signing identity"

	for command_name in go clang lipo codesign hdiutil plutil otool shasum xcode-select; do
		require_command "$command_name"
	done
	if [[ "$INSTALL_BUNDLE" == 1 ]]; then
		for command_name in pgrep ps ditto open; do
			require_command "$command_name"
		done
	fi
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

current_workmuch_pids() {
	local candidates
	local command_line
	local pid

	candidates=$(pgrep -f "$INSTALLED_EXECUTABLE" 2>/dev/null || true)
	for pid in $candidates; do
		command_line=$(ps -p "$pid" -o command= 2>/dev/null || true)
		if [[ "$command_line" == "$INSTALLED_EXECUTABLE" ||
			"$command_line" == "$INSTALLED_EXECUTABLE "* ]]; then
			printf '%s\n' "$pid"
		fi
	done
}

wait_for_workmuch_exit() {
	local attempts=$1
	local attempt

	for ((attempt = 0; attempt < attempts; attempt++)); do
		if [[ -z $(current_workmuch_pids) ]]; then
			return 0
		fi
		sleep "$INSTALL_POLL_INTERVAL_SECONDS"
	done
	[[ -z $(current_workmuch_pids) ]]
}

stop_installed_app() {
	local pids

	pids=$(current_workmuch_pids)
	if [[ -z "$pids" ]]; then
		return
	fi

	echo "[Stopping installed WorkMuch.app...]"
	env kill -TERM $pids 2>/dev/null || true
	if wait_for_workmuch_exit "$INSTALL_TERM_WAIT_ATTEMPTS"; then
		return
	fi

	pids=$(current_workmuch_pids)
	if [[ -z "$pids" ]]; then
		return
	fi

	echo "[Force stopping installed WorkMuch.app...]"
	env kill -KILL $pids 2>/dev/null || true
	if ! wait_for_workmuch_exit "$INSTALL_KILL_WAIT_ATTEMPTS"; then
		fail "installed WorkMuch.app did not stop"
	fi
}

install_built_app() {
	local app_path=$1
	local short_version=$2
	local bundle_version=$3

	stop_installed_app
	echo "[Installing $INSTALLED_APP...]"
	mkdir -p "$(dirname -- "$INSTALLED_APP")"
	rm -rf -- "$INSTALLED_APP"
	ditto "$app_path" "$INSTALLED_APP"
	verify_bundle "$INSTALLED_APP" "$short_version" "$bundle_version"

	echo "[Opening $INSTALLED_APP...]"
	open -a "$INSTALLED_APP"
	echo "Installed and opened $INSTALLED_APP"
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
	if [[ "$INSTALL_BUNDLE" == 1 ]]; then
		install_built_app "$app_path" "$short_version" "$bundle_version"
	fi
}

parse_args "$@"
build_release
