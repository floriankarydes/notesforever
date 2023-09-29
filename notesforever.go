package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Name:  "notesforever",
		Usage: "backup macOS Notes to a Git repository",
		Commands: []*cli.Command{
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   "initialize backup system",
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
				Usage:   "initialize backup system & set up backup service",
				Action:  Configure,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func Init(c *cli.Context) error {
	//TODO
	return nil
}

func Backup(c *cli.Context) error {
	//TODO
	return nil
}

func Restore(c *cli.Context) error {
	//TODO
	return nil
}

func Configure(c *cli.Context) error {
	//TODO
	return nil
}
