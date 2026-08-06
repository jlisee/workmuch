package tray

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"workmuch-go/internal/buildinfo"
)

const aboutMessage = "Current app and window title based activity tracker"
const aboutTitle = "About Workmuch"

func showAboutScreen() error {
	cmd, err := aboutCommand(runtime.GOOS, buildinfo.Version)
	if err != nil {
		return err
	}
	return cmd.Start()
}

func aboutCommand(goos string, version string) (*exec.Cmd, error) {
	message := aboutMessage + "\n\nVersion " + version
	switch goos {
	case "darwin":
		script := `display dialog "` + message + `" with title "` + aboutTitle + `" buttons {"OK"} default button "OK"`
		return exec.Command("osascript", "-e", script), nil
	case "linux":
		target, err := serveBrowserPage([]byte(renderAboutHTML(version)))
		if err != nil {
			return nil, err
		}
		return exec.Command("xdg-open", target), nil
	case "windows":
		target, err := ensureAboutHTML(version)
		if err != nil {
			return nil, err
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target), nil
	default:
		return nil, fmt.Errorf("about screen unsupported on %s", goos)
	}
}

func ensureAboutHTML(version string) (string, error) {
	path := filepath.Join(os.TempDir(), "workmuch-about.html")
	if err := os.WriteFile(path, []byte(renderAboutHTML(version)), 0o644); err != nil {
		return "", fmt.Errorf("write about page: %w", err)
	}
	return path, nil
}

func renderAboutHTML(version string) string {
	return `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + aboutTitle + `</title>
  <style>
    :root {
      color-scheme: light;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background: #f3f3f3;
    }
    main {
      width: min(520px, calc(100% - 3rem));
      padding: 2rem;
      border-radius: 14px;
      background: #ffffff;
      box-shadow: 0 12px 30px rgba(0, 0, 0, 0.12);
      text-align: center;
    }
    h1 {
      margin: 0 0 1rem;
      font-size: 1.5rem;
    }
    p {
      margin: 0;
      font-size: 1rem;
      line-height: 1.5;
    }
  </style>
</head>
<body>
  <main>
    <h1>Workmuch</h1>
    <p>` + aboutMessage + `</p>
    <p>Version ` + version + `</p>
  </main>
</body>
</html>`
}
