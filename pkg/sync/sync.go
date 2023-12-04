package sync

import (
	"os"
	"path/filepath"
	"time"

	"github.com/floriankarydes/notesforever/pkg/git"
	cp "github.com/otiai10/copy"
	"github.com/pkg/errors"
)

type Link struct {
	repo   *git.Repo
	srcDir string
}

const backupDirname = "backup"

func New(repo *git.Repo, srcDir string) (*Link, error) {
	m := &Link{
		repo:   repo,
		srcDir: srcDir,
	}
	return m, nil
}

func (m *Link) Backup() error {
	defer m.repo.Clean()

	// Clear destination directory.
	if err := os.RemoveAll(m.dstDir()); err != nil {
		return errors.Wrap(err, "failed to remove destination directory")
	}
	if err := os.MkdirAll(m.dstDir(), git.DirPerm); err != nil {
		return errors.Wrap(err, "failed to re-create destination directory")
	}

	// Copy files to destination directory.
	if err := cp.Copy(m.srcDir, m.dstDir()); err != nil {
		return errors.Wrap(err, "failed to copy directory")
	}

	// Push all changes.
	if err := m.repo.Push(); err != nil {
		return errors.Wrap(err, "failed to push changes")
	}

	return nil
}

func (m *Link) Restore() error {

	// Clear src directory.
	if err := os.Rename(m.srcDir, m.saveDir()); err != nil {
		return errors.Wrap(err, "failed to save source directory")
	}
	if err := os.MkdirAll(m.srcDir, git.DirPerm); err != nil {
		return errors.Wrap(err, "failed to re-create source directory")
	}

	// Copy files from Git repository.
	if err := cp.Copy(m.dstDir(), m.srcDir); err != nil {
		return errors.Wrap(err, "failed to copy directory")
	}

	return nil
}

func (m *Link) dstDir() string {
	return filepath.Join(m.repo.Dir(), backupDirname)
}

func (m *Link) saveDir() string {
	return "notesforever_NotesBackup_" + time.Now().Format("20060102150405")
}
