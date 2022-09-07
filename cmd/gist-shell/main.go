package main

import (
	"log"
	"os"

	"github.com/mick/ultragist/ultragist"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dbpath",
				Usage: "path to the database",
			},
			&cli.StringFlag{
				Name:    "command",
				Aliases: []string{"c"},
				Usage:   "command to run",
			},
		},
		Action: func(c *cli.Context) error {
			return ultragist.GistShell(c.String("command"))
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
