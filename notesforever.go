package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/floriankarydes/notesforever/pkg/git"
	"github.com/floriankarydes/notesforever/pkg/service"
	"github.com/floriankarydes/notesforever/pkg/sync"
	"github.com/urfave/cli/v2"
)

const (
	moduleName   = "notesforever"
	notesUserDir = "Library/Group Containers/group.com.apple.notes"
	gitUserDir   = "." + moduleName
)

func main() {

	app := &cli.App{
		Name:  "notesforever",
		Usage: "backup macOS Notes to a Git repository",
		Commands: []*cli.Command{
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   "initialize backup file system",
				Action:  Init,
			},
			{
				Name:    "backup",
				Aliases: []string{"b"},
				Usage:   "backup notes",
				Action:  Backup,
			},
			{
				Name:    "restore",
				Aliases: []string{"r"},
				Usage:   "restore notes",
				Action:  Restore,
			},
			{
				Name:    "configure",
				Aliases: []string{"c"},
				Usage:   "initialize backup file system & set up background service",
				Action:  Configure,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func Init(c *cli.Context) error {
	_, err := openSyncLink()
	if err != nil {
		return err
	}
	return nil
}

func Backup(c *cli.Context) error {
	link, err := openSyncLink()
	if err != nil {
		return err
	}
	if err := link.Backup(); err != nil {
		return err
	}
	return nil
}

func Restore(c *cli.Context) error {
	link, err := openSyncLink()
	if err != nil {
		return err
	}
	if err := link.Restore(); err != nil {
		return err
	}
	return nil
}

func Configure(c *cli.Context) error {
	if _, err := openSyncLink(); err != nil {
		return err
	}
	if err := service.RunEverydayAt(0, moduleName, "backup"); err != nil {
		return err
	}
	return nil
}

func openSyncLink() (*sync.Link, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	gitDir := filepath.Join(homeDir, gitUserDir)
	syncDir := filepath.Join(homeDir, notesUserDir)
	repo, err := git.Open(gitDir, "")
	if err != nil {
		return nil, err
	}
	return sync.New(repo, syncDir)
}
