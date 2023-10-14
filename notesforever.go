package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/floriankarydes/notesforever/pkg/git"
	"github.com/floriankarydes/notesforever/pkg/service"
	"github.com/floriankarydes/notesforever/pkg/sync"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/process"
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
	log.Println("starting backup...")
	link, err := openSyncLink()
	if err != nil {
		return err
	}
	if err := link.Backup(); err != nil {
		return err
	}
	log.Println("backup completed")
	return nil
}

func Restore(c *cli.Context) error {
	log.Println("restoring...")
	link, err := openSyncLink()
	if err != nil {
		return err
	}
	if err := closeNotesApp(c.Context); err != nil {
		return errors.Wrap(err, "failed to close Notes app")
	}
	if err := link.Restore(); err != nil {
		return err
	}
	log.Println("restored")
	return nil
}

func Configure(c *cli.Context) error {
	log.Println("configuring...")
	if _, err := openSyncLink(); err != nil {
		return err
	}
	if err := service.RunEverydayAt(0, moduleName, "backup"); err != nil {
		return err
	}
	log.Println("configured successfully; make sure you give Full Disk Access to notesforever in System Preferences > Security & Privacy > Privacy > Full Disk Access")
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

const notesAppName = "Notes"

func closeNotesApp(ctx context.Context) error {
	log.Println("cannot close Notes app automatically, please close it manually before running this command")
	return nil
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return errors.Wrap(err, "cannot get processes list")
	}
	for _, p := range processes {
		n, err := p.NameWithContext(ctx)
		if err != nil {
			return errors.Wrap(err, "cannot get process name")
		}
		if n == notesAppName {
			err := p.SendSignalWithContext(ctx, syscall.SIGINT)
			if err != nil {
				return errors.Wrap(err, "cannot stop process")
			}
		}
	}
	return nil
}
