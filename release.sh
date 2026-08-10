#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "$SCRIPT_DIR"

export GIT_TERMINAL_PROMPT=0
export GOCACHE="${GOCACHE:-$SCRIPT_DIR/.gocache}"
mkdir -p "$GOCACHE"

usage() {
	cat >&2 <<EOF
usage:
  $0
  $0 --local <linux/amd64|linux/arm64>
  $0 --local darwin/universal [--install]
EOF
}

run_checks() {
	echo "[Running tests...]"
	./test.sh

	echo "[Running lint...]"
	./lint.sh
}

build_local_deb() {
	local platform=$1
	local architecture
	local other_architecture
	local version
	local artifact
	local other_artifact

	case "$platform" in
	linux/amd64)
		architecture=amd64
		other_architecture=arm64
		;;
	linux/arm64)
		architecture=arm64
		other_architecture=amd64
		;;
	*)
		echo "error: unsupported local platform \"$platform\"; use linux/amd64, linux/arm64, or darwin/universal" >&2
		exit 2
		;;
	esac

	run_checks
	version=$(go run ./cmd/workmuch-version --format version)
	if [[ -n "$(git status --porcelain=v1)" ]]; then
		version="${version}.dirty"
	fi

	echo "[Building workmuch $version for $platform...]"
	WORKMUCH_VERSION="$version" \
		go run github.com/goreleaser/goreleaser/v2@v2.17.1 \
		release --snapshot --clean

	artifact="dist/workmuch_${version}_${architecture}.deb"
	if [[ ! -f "$artifact" ]]; then
		echo "error: GoReleaser did not create $artifact" >&2
		exit 1
	fi
	other_artifact="dist/workmuch_${version}_${other_architecture}.deb"
	if [[ -f "$other_artifact" ]]; then
		rm -- "$other_artifact"
	fi
	(
		cd dist
		sha256sum "$(basename "$artifact")" >checksums.txt
	)
	echo "Built $artifact"
}

if [[ $# -gt 0 ]]; then
	if [[ $# -lt 2 || $1 != --local ]]; then
		usage
		exit 2
	fi
	if [[ $2 == darwin/universal ]]; then
		if [[ $# -gt 3 || (${3:-} != "" && ${3:-} != --install) ]]; then
			usage
			exit 2
		fi
		if [[ $# -eq 3 ]]; then
			packaging/macos/release.sh --install
		else
			packaging/macos/release.sh
		fi
		exit 0
	fi
	if [[ $# -ne 2 ]]; then
		usage
		exit 2
	fi
	build_local_deb "$2"
	exit 0
fi

if [[ -n "$(git status --porcelain=v1)" ]]; then
	echo "error: the worktree must be clean before releasing" >&2
	exit 1
fi

if ! git remote get-url origin >/dev/null 2>&1; then
	echo "error: the origin remote is not configured" >&2
	exit 1
fi

echo "[Fetching origin branches and tags...]"
git fetch --prune origin '+refs/heads/*:refs/remotes/origin/*'
git fetch --tags origin
git remote set-head origin --auto >/dev/null 2>&1 || true

run_checks

VERSION=$(go run ./cmd/workmuch-version --format version)
TAG=$(go run ./cmd/workmuch-version --format tag)
MAIN_BASE=$(go run ./cmd/workmuch-version --format base)
RELEASE_COMMIT=$(go run ./cmd/workmuch-version --format head)

validate_existing_tag() {
	local tagged_commit
	local recorded_base

	go run ./cmd/workmuch-version validate-tag "$TAG" >/dev/null
	tagged_commit=$(git rev-parse --verify "refs/tags/$TAG^{commit}")
	if [[ "$tagged_commit" != "$RELEASE_COMMIT" ]]; then
		echo "error: $TAG points to $tagged_commit, expected $RELEASE_COMMIT" >&2
		exit 1
	fi
	recorded_base=$(
		git for-each-ref --format='%(contents)' "refs/tags/$TAG" |
			sed -n 's/^WorkMuch-Main-Base: //p'
	)
	if [[ "$recorded_base" != "$MAIN_BASE" ]]; then
		echo "error: $TAG records main base $recorded_base, expected $MAIN_BASE" >&2
		exit 1
	fi
}

if git show-ref --verify --quiet "refs/tags/$TAG"; then
	echo "[Using existing identical tag $TAG...]"
	validate_existing_tag
else
	echo "[Creating annotated tag $TAG...]"
	git tag -a "$TAG" "$RELEASE_COMMIT" \
		-m "WorkMuch $VERSION" \
		-m "WorkMuch-Main-Base: $MAIN_BASE"
fi

remote_tag=$(git ls-remote --tags origin "refs/tags/$TAG" "refs/tags/$TAG^{}")
if [[ -n "$remote_tag" ]]; then
	validate_existing_tag
	remote_commit=$(
		awk -v peeled="refs/tags/$TAG^{}" '$2 == peeled { print $1 }' <<<"$remote_tag"
	)
	if [[ -z "$remote_commit" ]]; then
		echo "error: origin already has a non-annotated or conflicting $TAG" >&2
		exit 1
	fi
	if [[ "$remote_commit" != "$RELEASE_COMMIT" ]]; then
		echo "error: origin $TAG points to $remote_commit, expected $RELEASE_COMMIT" >&2
		exit 1
	fi
	echo "Tag $TAG is already on origin; the release can be retried safely."
else
	echo "[Pushing $TAG...]"
	git push origin "refs/tags/$TAG"
	echo "Pushed $TAG; GitHub Actions will publish the Debian packages."
fi
