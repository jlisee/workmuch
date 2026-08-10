package macosapp

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

const (
	InstalledBundlePath     = "/Applications/WorkMuch.app"
	InstalledExecutablePath = InstalledBundlePath + "/Contents/MacOS/workmuch"
)

type LoginItemState string

const (
	LoginItemNotRegistered    LoginItemState = "not_registered"
	LoginItemEnabled          LoginItemState = "enabled"
	LoginItemRequiresApproval LoginItemState = "requires_approval"
	LoginItemNotFound         LoginItemState = "not_found"
	LoginItemUnsupported      LoginItemState = "unsupported"
	LoginItemNotApplicable    LoginItemState = "not_applicable"
)

type LoginItem interface {
	Status() (LoginItemState, error)
	Register() error
}

type Accessibility interface {
	IsTrusted() (bool, error)
	Prompt() error
}

type MoveDialog interface {
	Show() error
}

type LaunchEnvironment struct {
	Platform       string
	ExecutablePath string
}

type LaunchDependencies struct {
	LoginItem     LoginItem
	Accessibility Accessibility
	MoveDialog    MoveDialog
}

func PrepareBundledTrayLaunch(env LaunchEnvironment, deps LaunchDependencies, logger *log.Logger) (bool, error) {
	if strings.TrimSpace(env.Platform) != "darwin" || !isBundledExecutable(env.ExecutablePath) {
		return true, nil
	}
	if filepath.Clean(env.ExecutablePath) != InstalledExecutablePath {
		if deps.MoveDialog == nil {
			return false, fmt.Errorf("show move-to-Applications dialog: dialog is unavailable")
		}
		if err := deps.MoveDialog.Show(); err != nil {
			return false, fmt.Errorf("show move-to-Applications dialog: %w", err)
		}
		return false, nil
	}

	if logger == nil {
		logger = log.Default()
	}
	configureLoginItem(deps.LoginItem, logger)
	promptForAccessibility(deps.Accessibility, logger)
	return true, nil
}

func isBundledExecutable(path string) bool {
	cleaned := filepath.Clean(path)
	suffix := filepath.Join("WorkMuch.app", "Contents", "MacOS", "workmuch")
	return cleaned == suffix || strings.HasSuffix(cleaned, string(filepath.Separator)+suffix)
}

func configureLoginItem(loginItem LoginItem, logger *log.Logger) {
	if loginItem == nil {
		logger.Print("login item unavailable")
		return
	}
	state, err := loginItem.Status()
	if err != nil {
		logger.Printf("login item status failed: %v", err)
		return
	}
	switch state {
	case LoginItemNotRegistered:
		if err := loginItem.Register(); err != nil {
			logger.Printf("login item registration failed: %v", err)
		}
	case LoginItemRequiresApproval:
		logger.Print("login item requires user approval in System Settings")
	case LoginItemEnabled:
		return
	case LoginItemNotFound, LoginItemUnsupported, LoginItemNotApplicable:
		logger.Printf("login item status: %s", state)
	default:
		logger.Printf("login item status: %s", state)
	}
}

func promptForAccessibility(accessibility Accessibility, logger *log.Logger) {
	if accessibility == nil {
		logger.Print("accessibility permission check unavailable")
		return
	}
	trusted, err := accessibility.IsTrusted()
	if err != nil {
		logger.Printf("accessibility permission check failed: %v", err)
		return
	}
	if trusted {
		return
	}
	if err := accessibility.Prompt(); err != nil {
		logger.Printf("accessibility permission prompt failed: %v", err)
	}
}

func mapLoginItemStatus(raw int) LoginItemState {
	switch raw {
	case 0:
		return LoginItemNotRegistered
	case 1:
		return LoginItemEnabled
	case 2:
		return LoginItemRequiresApproval
	case 3:
		return LoginItemNotFound
	default:
		return LoginItemUnsupported
	}
}
