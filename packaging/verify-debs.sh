#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
	echo "usage: $0 <version> [dist-directory]" >&2
	exit 2
fi

VERSION=$1
DIST_DIR=${2:-dist}
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
REPOSITORY_DIR=$(cd -- "$SCRIPT_DIR/.." &>/dev/null && pwd)
DIST_DIR=$(cd -- "$DIST_DIR" &>/dev/null && pwd)

dpkg --validate-version "$VERSION"

mapfile -t packages < <(find "$DIST_DIR" -maxdepth 1 -type f -name '*.deb' -printf '%f\n' | sort)
expected_packages=(
	"workmuch_${VERSION}_amd64.deb"
	"workmuch_${VERSION}_arm64.deb"
)
if [[ "${packages[*]}" != "${expected_packages[*]}" ]]; then
	echo "error: unexpected Debian artifacts: ${packages[*]}" >&2
	exit 1
fi

if [[ $(wc -l <"$DIST_DIR/checksums.txt") -ne 2 ]]; then
	echo "error: checksums.txt must contain exactly two artifacts" >&2
	exit 1
fi
mapfile -t checksum_packages < <(awk '{print $2}' "$DIST_DIR/checksums.txt" | sort)
[[ "${checksum_packages[*]}" == "${expected_packages[*]}" ]]
[[ $(grep -Ec '^[0-9a-f]{64}  [^ ]+$' "$DIST_DIR/checksums.txt") -eq 2 ]]
(cd "$DIST_DIR" && sha256sum --check checksums.txt)

for architecture in amd64 arm64; do
	package="$DIST_DIR/workmuch_${VERSION}_${architecture}.deb"
	[[ $(dpkg-deb --field "$package" Package) == workmuch ]]
	[[ $(dpkg-deb --field "$package" Version) == "$VERSION" ]]
	[[ $(dpkg-deb --field "$package" Architecture) == "$architecture" ]]
	[[ $(dpkg-deb --field "$package" Depends) == "init-system-helpers (>= 1.52)" ]]

	contents=$(dpkg-deb --contents "$package")
	grep -Eq '^-rwxr-xr-x .* \./usr/bin/workmuch$' <<<"$contents"
	grep -Eq '^-rw-r--r-- .* \./usr/lib/systemd/user/workmuch.service$' <<<"$contents"
	for document in README changelog.Debian.gz copyright service.md; do
		grep -Eq "^-rw-r--r-- .* \\./usr/share/doc/workmuch/${document}$" <<<"$contents"
	done

	extract_dir=$(mktemp -d /tmp/workmuch-package.XXXXXX)
	control_dir=$(mktemp -d /tmp/workmuch-control.XXXXXX)
	dpkg-deb --extract "$package" "$extract_dir"
	dpkg-deb --control "$package" "$control_dir"

	cmp "$REPOSITORY_DIR/packaging/systemd/workmuch.service" \
		"$extract_dir/usr/lib/systemd/user/workmuch.service"
	cmp "$REPOSITORY_DIR/packaging/debian/postinstall" "$control_dir/postinst"
	cmp "$REPOSITORY_DIR/packaging/debian/preremove" "$control_dir/prerm"
	cmp "$REPOSITORY_DIR/packaging/debian/postremove" "$control_dir/postrm"
	[[ $(stat -c '%a' "$control_dir/postinst") == 755 ]]
	[[ $(stat -c '%a' "$control_dir/prerm") == 755 ]]
	[[ $(stat -c '%a' "$control_dir/postrm") == 755 ]]
	gzip --test "$extract_dir/usr/share/doc/workmuch/changelog.Debian.gz"

	binary_info=$(file "$extract_dir/usr/bin/workmuch")
	grep -Fq 'statically linked' <<<"$binary_info"
	if [[ "$architecture" == amd64 ]]; then
		grep -Fq 'x86-64' <<<"$binary_info"
		actual_version=$("$extract_dir/usr/bin/workmuch" --version)
		[[ "$actual_version" == "workmuch $VERSION" ]]
	else
		grep -Fq 'ARM aarch64' <<<"$binary_info"
	fi
done

if find "$DIST_DIR" -maxdepth 1 -type f \
	\( -name '*.dmg' -o -name '*.tar.gz' -o -name '*.zip' -o -name 'workmuch' \) |
	grep -q .; then
	echo "error: found an unsupported loose, archive, or macOS artifact" >&2
	exit 1
fi

echo "Verified workmuch $VERSION Debian artifacts."
