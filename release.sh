#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "$SCRIPT_DIR"

export GIT_TERMINAL_PROMPT=0
export GOCACHE="${GOCACHE:-$SCRIPT_DIR/.gocache}"
mkdir -p "$GOCACHE"

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

echo "[Running tests...]"
./test_go.sh

echo "[Running lint...]"
./lint.sh

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
