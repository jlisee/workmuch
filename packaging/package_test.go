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
	assert.Contains(t, string(output), "linux/amd64, linux/arm64, or darwin/universal")
}

func TestMacOSBundleTemplateHasRequiredLayoutAndPlistKeys(t *testing.T) {
	t.Parallel()

	plist := readPackageFile(t, "macos/WorkMuch.app/Contents/Info.plist")
	assert.Contains(t, plist, "<key>CFBundleIdentifier</key>\n\t<string>com.jlisee.workmuch</string>")
	assert.Contains(t, plist, "<key>CFBundleExecutable</key>\n\t<string>workmuch</string>")
	assert.Contains(t, plist, "<key>CFBundlePackageType</key>\n\t<string>APPL</string>")
	assert.Contains(t, plist, "<key>LSUIElement</key>\n\t<true/>")
	assert.Contains(t, plist, "<key>LSMinimumSystemVersion</key>\n\t<string>13.0</string>")
	assert.Contains(t, plist, "__WORKMUCH_SHORT_VERSION__")
	assert.Contains(t, plist, "__WORKMUCH_BUNDLE_VERSION__")

	for _, path := range []string{
		"macos/WorkMuch.app/Contents/MacOS/workmuch",
		"macos/WorkMuch.app/Contents/Resources/WorkMuch.icns",
		"macos/release.sh",
	} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.False(t, info.IsDir())
	}

	icon, err := os.ReadFile("macos/WorkMuch.app/Contents/Resources/WorkMuch.icns")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(icon), 8)
	assert.Equal(t, "icns", string(icon[:4]))
}

func TestMacOSReleaseScriptBuildsSignsAndVerifiesUniversalDMG(t *testing.T) {
	t.Parallel()

	script := readPackageFile(t, "macos/release.sh")
	assert.Contains(t, script, "GOARCH=\"$architecture\"")
	assert.Contains(t, script, "CGO_ENABLED=1")
	assert.Contains(t, script, "MACOSX_DEPLOYMENT_TARGET=13.0")
	assert.Contains(t, script, "-trimpath")
	assert.Contains(t, script, "lipo -create")
	assert.Contains(t, script, "codesign --force")
	assert.Contains(t, script, "--timestamp=none")
	assert.Contains(t, script, "codesign --verify --deep --strict")
	assert.Contains(t, script, "codesign -d -r-")
	assert.Contains(t, script, "hdiutil create")
	assert.Contains(t, script, "-format UDZO")
	assert.Contains(t, script, "hdiutil verify")
	assert.Contains(t, script, "hdiutil attach -readonly")
	assert.Contains(t, script, "WorkMuch_${version}_universal.dmg")
	assert.Contains(t, script, "checksums.txt")
	assert.NotContains(t, script, "spctl")
}

func TestMacOSReleaseRejectsWrongHostBeforeRunningChecks(t *testing.T) {
	repositoryDir, err := filepath.Abs("..")
	require.NoError(t, err)
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "uname"), `#!/usr/bin/env bash
printf 'Linux\n'
`)

	command := exec.Command("bash", filepath.Join(repositoryDir, "release.sh"), "--local", "darwin/universal")
	command.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"WORKMUCH_CODESIGN_IDENTITY=Test Identity",
	)
	output, err := command.CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), "must be built on macOS")
	assert.NotContains(t, string(output), "[Running tests...]")
}

func TestMacOSReleaseRequiresSigningIdentityBeforeRunningChecks(t *testing.T) {
	repositoryDir, err := filepath.Abs("..")
	require.NoError(t, err)
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "uname"), `#!/usr/bin/env bash
printf 'Darwin\n'
`)

	command := exec.Command("bash", filepath.Join(repositoryDir, "release.sh"), "--local", "darwin/universal")
	command.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"WORKMUCH_CODESIGN_IDENTITY=",
	)
	output, err := command.CombinedOutput()

	require.Error(t, err)
	assert.Contains(t, string(output), "WORKMUCH_CODESIGN_IDENTITY")
	assert.NotContains(t, string(output), "[Running tests...]")
}

func TestMacOSReleaseStopsAfterFailedArchitectureBuild(t *testing.T) {
	scriptPath, binDir, commandLog := setupFakeMacRelease(t)
	command := exec.Command("bash", scriptPath)
	command.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"COMMAND_LOG="+commandLog,
		"FAKE_HDIUTIL_STATE="+filepath.Join(t.TempDir(), "hdiutil-state"),
		"WORKMUCH_CODESIGN_IDENTITY=Test Identity",
		"FAIL_ARCH=arm64",
	)

	output, err := command.CombinedOutput()

	require.Error(t, err, string(output))
	commands := readTestFile(t, commandLog)
	assert.Contains(t, commands, "go:build")
	assert.Contains(t, commands, "arch=arm64")
	assert.NotContains(t, commands, "codesign:")
	assert.NotContains(t, commands, "hdiutil:")
}

func TestMacOSReleaseStopsWhenSignatureVerificationFails(t *testing.T) {
	scriptPath, binDir, commandLog := setupFakeMacRelease(t)
	command := exec.Command("bash", scriptPath)
	command.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"COMMAND_LOG="+commandLog,
		"FAKE_HDIUTIL_STATE="+filepath.Join(t.TempDir(), "hdiutil-state"),
		"WORKMUCH_CODESIGN_IDENTITY=Test Identity",
		"FAIL_CODESIGN_VERIFY=1",
	)

	output, err := command.CombinedOutput()

	require.Error(t, err, string(output))
	commands := readTestFile(t, commandLog)
	assert.Contains(t, commands, "arch=arm64")
	assert.Contains(t, commands, "arch=amd64")
	assert.Contains(t, commands, "codesign:--force --sign Test Identity --timestamp=none")
	assert.Contains(t, commands, "codesign:--verify --deep --strict")
	assert.NotContains(t, commands, "hdiutil:create")
}

func TestMacOSReleaseWritesVersionedDMGAndChecksum(t *testing.T) {
	scriptPath, binDir, commandLog := setupFakeMacRelease(t)
	hdiutilState := filepath.Join(t.TempDir(), "hdiutil-state")
	command := exec.Command("bash", scriptPath)
	command.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"COMMAND_LOG="+commandLog,
		"FAKE_HDIUTIL_STATE="+hdiutilState,
		"WORKMUCH_CODESIGN_IDENTITY=Test Identity",
	)

	output, err := command.CombinedOutput()

	require.NoError(t, err, string(output))
	repositoryDir := filepath.Dir(filepath.Dir(filepath.Dir(scriptPath)))
	distDir := filepath.Join(repositoryDir, "dist", "macos")
	artifact := "WorkMuch_" + testPackageVersion + "_universal.dmg"
	_, err = os.Stat(filepath.Join(distDir, artifact))
	require.NoError(t, err)
	checksum := readTestFile(t, filepath.Join(distDir, "checksums.txt"))
	assert.Contains(t, checksum, artifact)
	commands := readTestFile(t, commandLog)
	assert.Contains(t, commands, "lipo:-create")
	assert.Contains(t, commands, "hdiutil:verify")
	assert.Contains(t, commands, "hdiutil:attach -readonly")
}

func setupFakeMacRelease(t *testing.T) (string, string, string) {
	t.Helper()

	repositoryDir := t.TempDir()
	macosDir := filepath.Join(repositoryDir, "packaging", "macos")
	appContents := filepath.Join(macosDir, "WorkMuch.app", "Contents")
	require.NoError(t, os.MkdirAll(filepath.Join(appContents, "MacOS"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(appContents, "Resources"), 0o755))
	copyTestFile(t, "macos/release.sh", filepath.Join(macosDir, "release.sh"), 0o755)
	copyTestFile(t, "macos/WorkMuch.app/Contents/Info.plist", filepath.Join(appContents, "Info.plist"), 0o644)
	copyTestFile(t, "macos/WorkMuch.app/Contents/MacOS/workmuch", filepath.Join(appContents, "MacOS", "workmuch"), 0o755)
	copyTestFile(t, "macos/WorkMuch.app/Contents/Resources/WorkMuch.icns", filepath.Join(appContents, "Resources", "WorkMuch.icns"), 0o644)

	commandLog := filepath.Join(repositoryDir, "commands.log")
	writeExecutable(t, filepath.Join(repositoryDir, "test.sh"), `#!/usr/bin/env bash
printf 'test\n' >>"$COMMAND_LOG"
`)
	writeExecutable(t, filepath.Join(repositoryDir, "lint.sh"), `#!/usr/bin/env bash
printf 'lint\n' >>"$COMMAND_LOG"
`)

	binDir := filepath.Join(repositoryDir, "bin")
	require.NoError(t, os.Mkdir(binDir, 0o755))
	writeExecutable(t, filepath.Join(binDir, "uname"), `#!/usr/bin/env bash
printf 'Darwin\n'
`)
	writeExecutable(t, filepath.Join(binDir, "xcode-select"), `#!/usr/bin/env bash
exit 0
`)
	writeExecutable(t, filepath.Join(binDir, "clang"), `#!/usr/bin/env bash
printf 'clang:%s\n' "$*" >>"$COMMAND_LOG"
exit 0
`)
	writeExecutable(t, filepath.Join(binDir, "git"), `#!/usr/bin/env bash
printf 'git:%s\n' "$*" >>"$COMMAND_LOG"
if [[ "$*" == "status --porcelain=v1" ]]; then
	exit 0
fi
exit 90
`)
	writeExecutable(t, filepath.Join(binDir, "go"), `#!/usr/bin/env bash
printf 'go:%s|arch=%s\n' "$*" "${GOARCH:-}" >>"$COMMAND_LOG"
if [[ "$*" == "env CGO_ENABLED" ]]; then
	printf '1\n'
	exit 0
fi
if [[ "$*" == "run ./cmd/workmuch-version --format version" ]]; then
	printf '%s\n' "`+testPackageVersion+`"
	exit 0
fi
if [[ "$*" == "run ./cmd/workmuch-version --format plist-short" ]]; then
	printf '2026.8.6\n'
	exit 0
fi
if [[ "$*" == "run ./cmd/workmuch-version --format plist-build" ]]; then
	printf '52.3.0\n'
	exit 0
fi
if [[ "$1" == "build" ]]; then
	if [[ -n "${FAIL_ARCH:-}" && "${GOARCH:-}" == "$FAIL_ARCH" ]]; then
		exit 41
	fi
	output=''
	previous=''
	for argument in "$@"; do
		if [[ "$previous" == '-o' ]]; then
			output=$argument
			break
		fi
		previous=$argument
	done
	touch "$output"
	exit 0
fi
exit 91
`)
	writeExecutable(t, filepath.Join(binDir, "lipo"), `#!/usr/bin/env bash
printf 'lipo:%s\n' "$*" >>"$COMMAND_LOG"
if [[ "$1" == '-archs' ]]; then
	printf 'arm64 x86_64\n'
	exit 0
fi
if [[ "$1" == '-create' ]]; then
	previous=''
	for argument in "$@"; do
		if [[ "$previous" == '-output' ]]; then
			touch "$argument"
			exit 0
		fi
		previous=$argument
	done
fi
exit 92
`)
	writeExecutable(t, filepath.Join(binDir, "codesign"), `#!/usr/bin/env bash
printf 'codesign:%s\n' "$*" >>"$COMMAND_LOG"
if [[ "$1" == '--verify' && "${FAIL_CODESIGN_VERIFY:-}" == 1 ]]; then
	exit 42
fi
exit 0
`)
	writeExecutable(t, filepath.Join(binDir, "plutil"), `#!/usr/bin/env bash
printf 'plutil:%s\n' "$*" >>"$COMMAND_LOG"
if [[ "$1" == '-lint' ]]; then
	exit 0
fi
case "$2" in
CFBundleIdentifier) printf 'com.jlisee.workmuch\n' ;;
CFBundleExecutable) printf 'workmuch\n' ;;
CFBundleIconFile) printf 'WorkMuch\n' ;;
CFBundlePackageType) printf 'APPL\n' ;;
CFBundleShortVersionString) printf '2026.8.6\n' ;;
CFBundleVersion) printf '52.3.0\n' ;;
LSMinimumSystemVersion) printf '13.0\n' ;;
LSUIElement) printf 'true\n' ;;
*) exit 93 ;;
esac
`)
	writeExecutable(t, filepath.Join(binDir, "otool"), `#!/usr/bin/env bash
printf 'otool:%s\n' "$*" >>"$COMMAND_LOG"
printf '      cmd LC_BUILD_VERSION\n    minos 13.0\n'
`)
	writeExecutable(t, filepath.Join(binDir, "hdiutil"), `#!/usr/bin/env bash
printf 'hdiutil:%s\n' "$*" >>"$COMMAND_LOG"
case "$1" in
create)
	previous=''
	for argument in "$@"; do
		if [[ "$previous" == '-srcfolder' ]]; then
			printf '%s\n' "$argument" >"$FAKE_HDIUTIL_STATE"
		fi
		previous=$argument
	done
	touch "${@: -1}"
	;;
attach)
	previous=''
	for argument in "$@"; do
		if [[ "$previous" == '-mountpoint' ]]; then
			mountpoint=$argument
			break
		fi
		previous=$argument
	done
	source_dir=$(<"$FAKE_HDIUTIL_STATE")
	cp -R "$source_dir/WorkMuch.app" "$mountpoint/WorkMuch.app"
	;;
verify|detach) ;;
*) exit 94 ;;
esac
`)
	writeExecutable(t, filepath.Join(binDir, "shasum"), `#!/usr/bin/env bash
printf 'shasum:%s\n' "$*" >>"$COMMAND_LOG"
if [[ "$*" == *'--check'* ]]; then
	exit 0
fi
printf '%064d  %s\n' 0 "${@: -1}"
`)

	returnScript := filepath.Join(macosDir, "release.sh")
	return returnScript, binDir, commandLog
}

func copyTestFile(t *testing.T, source string, target string, mode os.FileMode) {
	t.Helper()
	contents, err := os.ReadFile(source)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(target, contents, mode))
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(contents)
}

func writeExecutable(t *testing.T, path string, contents string) {
	t.Helper()

	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o755))
}
