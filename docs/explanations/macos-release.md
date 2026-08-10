# macOS installation and local releases

WorkMuch supports macOS 13 Ventura and later. The local release command builds
one universal app for Apple Silicon and Intel, signs it with a maintainer-owned
self-signed certificate, and places it in a compressed disk image. It does not
upload macOS artifacts to GitHub.

The build is locally signed, not Developer ID signed or notarized. Gatekeeper
therefore will not approve it through `spctl` like a notarized public release.
Only install a disk image received through a trusted channel and compare its
checksum before opening it.

## Create and protect the signing identity

Create the identity once in Keychain Access:

1. Open **Keychain Access > Certificate Assistant > Create a Certificate**.
2. Give it a stable name such as `WorkMuch Local Code Signing`.
3. Choose **Self Signed Root** for the identity type and **Code Signing** for
   the certificate type. Select the option to override defaults if a longer
   validity period is needed.
4. Store the identity in the login keychain and confirm that macOS lists it:

   ```bash
   security find-identity -v -p codesigning
   ```

Set the certificate trust needed by the local Mac through Keychain Access if
`codesign` reports that the identity is not trusted. Export the certificate and
its private key together as an encrypted `.p12` file. Keep that file and its
password in separate, backed-up, access-controlled locations. Losing the
private key means future releases cannot retain the same signing identity.

Do not commit the certificate, private key, exported `.p12`, or its password.

## Build the universal disk image

The build requires macOS, a CGO-enabled Go toolchain, Xcode Command Line Tools,
and the signing identity in the current user's keychain:

```bash
export WORKMUCH_CODESIGN_IDENTITY="WorkMuch Local Code Signing"
./release.sh --local darwin/universal
```

The script runs `./test.sh` and `./lint.sh`, builds arm64 and amd64 executables
with a macOS 13 deployment target, combines them, completes and signs
`WorkMuch.app`, and verifies the bundle both before and after creating the DMG.
The checked-in `WorkMuch.icns` is derived from `assets/icon.svg`; normal release
builds reuse it and do not require an icon-generation tool.
It writes:

```text
dist/macos/WorkMuch_<version>_universal.dmg
dist/macos/checksums.txt
```

Local source changes append `.dirty` to the complete version embedded in the
binary and to the DMG name. The script verifies both `arm64` and `x86_64`, the
deployment target, plist metadata, signature, designated requirement, DMG, and
checksum. It intentionally does not run `spctl`, because this identity is not a
notarized Developer ID identity.

## Install and open WorkMuch

1. Verify `checksums.txt`, open the DMG, and drag `WorkMuch.app` onto its
   **Applications** link.
2. Eject the DMG. WorkMuch refuses to run from the disk image or any location
   other than `/Applications/WorkMuch.app`.
3. In Finder, Control-click `/Applications/WorkMuch.app` and choose **Open**.
   If macOS still blocks it, use **System Settings > Privacy & Security > Open
   Anyway**, then confirm that you trust this local build.

On its first installed launch, WorkMuch asks macOS for Accessibility access.
The prompt is asynchronous, so collection continues with partial data until
access is granted. Enable WorkMuch under **System Settings > Privacy & Security
> Accessibility**, then quit and reopen it to capture focused window titles.

WorkMuch also registers its main app under **System Settings > General > Login
Items**. If a user disables it there, its state becomes `requires_approval` and
WorkMuch will not force it back on. Checkout launches through `./run.sh`,
`doctor`, `--qa-console`, and `--no-tray` never register a Login Item or request
Accessibility automatically.

These behaviors use Apple's
[`SMAppService.mainAppService`](https://developer.apple.com/documentation/servicemanagement/smappservice/mainapp?language=objc)
and
[`AXIsProcessTrustedWithOptions`](https://developer.apple.com/documentation/applicationservices/1459186-axisprocesstrustedwithoptions?language=objc)
APIs.

## Update and uninstall

For an update, quit WorkMuch and replace `/Applications/WorkMuch.app` with the
new bundle. Keep the same bundle identifier, executable name, installation
path, certificate, and private key across releases; this gives macOS the stable
identity it needs to retain Accessibility and Login Item decisions. Confirm the
permissions after every update rather than assuming they were preserved.

To uninstall, quit WorkMuch, disable it under **System Settings > General >
Login Items**, and move `/Applications/WorkMuch.app` to the Trash. Work logs and
runtime diagnostics remain in `~/.workmuch`; remove that directory separately
only if its history is no longer needed.
