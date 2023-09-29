package macosnotes

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const notesUserDir = "Library/Group Containers/group.com.apple.notes/"

func NotesDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to get user home directory")
	}
	return filepath.Join(homeDir, notesUserDir), nil
}
