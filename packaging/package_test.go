package packaging

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPackageVersion      = "20260806.52.3+g0123456789ab"
	testDirtyPackageVersion = testPackageVersion + ".dirty"
)

func readPackageFile(t *testing.T, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Clean(relativePath))
	require.NoError(t, err)
	return string(content)
}

func TestUserServiceRunsLinuxTrayCollectorInGraphicalSession(t *testing.T) {
	t.Parallel()

	unit := readPackageFile(t, "systemd/workmuch.service")

	assert.Contains(t, unit, "WantedBy=graphical-session.target")
	assert.Contains(t, unit, "PartOf=graphical-session.target")
	assert.Contains(t, unit, "After=graphical-session.target")
	assert.Contains(t, unit, `ExecCondition=/bin/sh -c 'test -n "$DISPLAY"'`)
	assert.Contains(t, unit, "ExecStart=/usr/bin/workmuch --backend linux")
	assert.NotContains(t, unit, "--no-tray")
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
	assert.Contains(t, config, "src: docs/explanations/debian-service.md")
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

	documentation := readPackageFile(t, "../docs/explanations/debian-service.md")

	assert.Contains(t, documentation, "systemctl --user status workmuch.service")
	assert.Contains(t, documentation, "journalctl --user -u workmuch.service")
	assert.Contains(t, documentation, "systemctl --user mask workmuch.service")
	assert.Contains(t, documentation, "system tray")
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

func TestReleaseScriptMarksChangedLocalBuildDirty(t *testing.T) {
	t.Parallel()

	repositoryDir, err := filepath.Abs("..")
	require.NoError(t, err)
	testDir := t.TempDir()
	commandLog := filepath.Join(testDir, "commands.log")

	releaseScript, err := os.ReadFile(filepath.Join(repositoryDir, "release.sh"))
	require.NoError(t, err)
	writeExecutable(t, filepath.Join(testDir, "release.sh"), string(releaseScript))
	writeExecutable(t, filepath.Join(testDir, "test.sh"), `#!/usr/bin/env bash
printf 'test\n' >>"$COMMAND_LOG"
`)
	writeExecutable(t, filepath.Join(testDir, "lint.sh"), `#!/usr/bin/env bash
printf 'lint\n' >>"$COMMAND_LOG"
`)

	binDir := filepath.Join(testDir, "bin")
	require.NoError(t, os.Mkdir(binDir, 0o755))
	writeExecutable(t, filepath.Join(binDir, "git"), `#!/usr/bin/env bash
printf 'git:%s\n' "$*" >>"$COMMAND_LOG"
if [[ "$*" == "status --porcelain=v1" ]]; then
	printf ' M internal/app/runner.go\n'
	exit 0
fi
exit 99
`)
	writeExecutable(t, filepath.Join(binDir, "go"), `#!/usr/bin/env bash
printf 'go:%s|version=%s|arch=%s\n' "$*" "${WORKMUCH_VERSION:-}" "${WORKMUCH_ARCH:-}" >>"$COMMAND_LOG"
if [[ "$*" == "run ./cmd/workmuch-version --format version" ]]; then
	printf '%s\n' "`+testPackageVersion+`"
	exit 0
fi
if [[ "$1" == "run" && "$2" == "github.com/goreleaser/goreleaser/v2@v2.17.1" ]]; then
	mkdir -p dist
	touch "dist/workmuch_${WORKMUCH_VERSION}_amd64.deb"
	touch "dist/workmuch_${WORKMUCH_VERSION}_arm64.deb"
	exit 0
fi
exit 98
`)

	command := exec.Command("bash", filepath.Join(testDir, "release.sh"), "--local", "linux/arm64")
	command.Env = append(os.Environ(),
		"COMMAND_LOG="+commandLog,
		"PATH="+binDir+":"+os.Getenv("PATH"),
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Contains(t, string(output), "dist/workmuch_"+testDirtyPackageVersion+"_arm64.deb")

	commands, err := os.ReadFile(commandLog)
	require.NoError(t, err)
	commandLines := string(commands)
	assert.Contains(t, commandLines, "test\n")
	assert.Contains(t, commandLines, "lint\n")
	assert.Contains(t, commandLines,
		"go:run github.com/goreleaser/goreleaser/v2@v2.17.1 release --snapshot --clean|version="+testDirtyPackageVersion)
	assert.Contains(t, commandLines, "git:status --porcelain=v1")
	assert.NotContains(t, commandLines, "git:fetch")
	assert.NotContains(t, commandLines, "git:tag")
	assert.NotContains(t, commandLines, "git:push")

	entries, err := os.ReadDir(filepath.Join(testDir, "dist"))
	require.NoError(t, err)
	var debs []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".deb") {
			debs = append(debs, entry.Name())
		}
	}
	assert.Equal(t, []string{"workmuch_" + testDirtyPackageVersion + "_arm64.deb"}, debs)

	checksum, err := os.ReadFile(filepath.Join(testDir, "dist", "checksums.txt"))
	require.NoError(t, err)
	assert.Contains(t, string(checksum), "workmuch_"+testDirtyPackageVersion+"_arm64.deb")
	assert.NotContains(t, string(checksum), "_amd64.deb")
}

func TestReleaseScriptRejectsUnsupportedLocalPlatform(t *testing.T) {
	t.Parallel()

	repositoryDir, err := filepath.Abs("..")
	require.NoError(t, err)
	command := exec.Command("bash", filepath.Join(repositoryDir, "release.sh"), "--local", "windows/amd64")
	output, err := command.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(output), `unsupported local platform "windows/amd64"`)
	assert.Contains(t, string(output), "linux/amd64 or linux/arm64")
}

func writeExecutable(t *testing.T, path string, contents string) {
	t.Helper()

	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o755))
}
