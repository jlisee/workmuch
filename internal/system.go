package internal

import (
	"os"
	"os/user"
	"strings"
)

var lookupCurrentUser = user.Current

func Username() (string, error) {
	if value := strings.TrimSpace(os.Getenv("USER")); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(os.Getenv("USERNAME")); value != "" {
		return value, nil
	}

	currentUser, err := lookupCurrentUser()
	if err != nil {
		return "", err
	}
	return currentUser.Username, nil
}
