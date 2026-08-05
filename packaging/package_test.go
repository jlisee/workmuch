package packaging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readPackageFile(t *testing.T, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Clean(relativePath))
	require.NoError(t, err)
	return string(content)
}

func TestUserServiceRunsLinuxCollectorInGraphicalSession(t *testing.T) {
	t.Parallel()

	unit := readPackageFile(t, "systemd/workmuch.service")

	assert.Contains(t, unit, "WantedBy=graphical-session.target")
	assert.Contains(t, unit, "PartOf=graphical-session.target")
	assert.Contains(t, unit, "After=graphical-session.target")
	assert.Contains(t, unit, `ExecCondition=/bin/sh -c 'test -n "$DISPLAY"'`)
	assert.Contains(t, unit, "ExecStart=/usr/bin/workmuch --no-tray --backend linux")
	assert.Contains(t, unit, "Restart=on-failure")
	assert.Contains(t, unit, "RestartSec=3s")
	assert.Contains(t, unit, "StartLimitIntervalSec=30s")
	assert.Contains(t, unit, "StartLimitBurst=5")
	assert.NotContains(t, unit, "User=")
	assert.NotContains(t, unit, "HOME=")
}

func TestMaintainerScriptsFollowSystemdUserConventions(t *testing.T) {
	t.Parallel()

	postinstall := readPackageFile(t, "debian/postinstall")
	assert.Contains(t, postinstall, "deb-systemd-helper --user unmask workmuch.service")
	assert.Contains(t, postinstall, "deb-systemd-helper --quiet --user was-enabled workmuch.service")
	assert.Contains(t, postinstall, "deb-systemd-helper --user enable workmuch.service")
	assert.Contains(t, postinstall, "deb-systemd-invoke --user restart workmuch.service")

	preremove := readPackageFile(t, "debian/preremove")
	assert.Contains(t, preremove, "deb-systemd-invoke --user stop workmuch.service")

	postremove := readPackageFile(t, "debian/postremove")
	assert.Contains(t, postremove, "deb-systemd-helper --user mask workmuch.service")
	assert.Contains(t, postremove, "deb-systemd-helper --user purge workmuch.service")
	assert.Contains(t, postremove, "deb-systemd-helper --user unmask workmuch.service")

	for _, script := range []string{postinstall, preremove, postremove} {
		assert.NotContains(t, script, ".workmuch")
		assert.NotContains(t, script, "rm -")
	}
}

func TestGoReleaserBuildsOnlyLinuxDebPackages(t *testing.T) {
	t.Parallel()

	config := readPackageFile(t, "../.goreleaser.yaml")

	assert.Contains(t, config, "CGO_ENABLED=0")
	assert.Contains(t, config, "goos:\n      - linux")
	assert.Contains(t, config, "id: workmuch-amd64")
	assert.Contains(t, config, "id: workmuch-arm64")
	assert.Contains(t, config, "goarch:\n      - amd64")
	assert.Contains(t, config, "goarch:\n      - arm64")
	assert.Contains(t, config, "workmuch-go/internal/buildinfo.Version={{ .Version }}")
	assert.Contains(t, config, "formats:\n      - deb")
	assert.Contains(t, config, "changelog: packaging/debian/changelog.yaml")
	assert.Contains(t, config, "dst: /usr/lib/systemd/user/workmuch.service")
	assert.Contains(t, config, "formats:\n      - none")
	assert.NotContains(t, config, "darwin")
	assert.NotContains(t, config, "dmg")
	assert.NotContains(t, config, "notar")
	assert.Contains(t, config, "id: workmuch-deb-amd64")
	assert.Contains(t, config, "id: workmuch-deb-arm64")
	assert.Contains(t, config, "unknown-field Architecture-Variant")
}

func TestServiceDocumentationExplainsOperationsAndDisplaySupport(t *testing.T) {
	t.Parallel()

	documentation := readPackageFile(t, "../docs/debian-service.md")

	assert.Contains(t, documentation, "systemctl --user status workmuch.service")
	assert.Contains(t, documentation, "journalctl --user -u workmuch.service")
	assert.Contains(t, documentation, "systemctl --user mask workmuch.service")
	assert.Contains(t, documentation, "X11")
	assert.Contains(t, documentation, "XWayland")
	assert.Contains(t, documentation, "pure Wayland")
}

func TestPackageVerifierChecksBothArchitecturesAndChecksums(t *testing.T) {
	t.Parallel()

	verifier := readPackageFile(t, "verify-debs.sh")

	assert.Contains(t, verifier, "workmuch_${VERSION}_amd64.deb")
	assert.Contains(t, verifier, "workmuch_${VERSION}_arm64.deb")
	assert.Contains(t, verifier, "sha256sum --check checksums.txt")
	assert.Contains(t, verifier, "dpkg-deb --field \"$package\" Version")
	assert.Contains(t, verifier, "workmuch.service")
	assert.Contains(t, verifier, "changelog.Debian.gz")
	assert.Contains(t, verifier, "postinst")
	assert.Contains(t, verifier, "statically linked")
}
