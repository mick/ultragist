package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/mick/ultragist/ultragist"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "keys",
				Usage: "commands for dealing with keys",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a key",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "keyfile",
								Usage:    "path to public key file",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "userid",
								Usage:    "userid of the key",
								Required: true,
							},
						},
						Action: func(c *cli.Context) error {
							keyfile := c.String("keyfile")
							userid := c.String("userid")

							publickeycontent, err := ioutil.ReadFile(keyfile)
							if err != nil {
								return err
							}

							return ultragist.WriteKey(publickeycontent, userid)

						},
					},
					{
						Name:  "authorizedkeys",
						Usage: "output authorized keys file based on fingerprint",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "fingerprint",
								Usage:    "fingerprint of the key",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "dbpath",
								Usage: "path to the database",
							},
						},
						Action: func(c *cli.Context) error {
							dbpath := c.String("dbpath")
							os.Setenv("DBPATH", dbpath)
							fingerprint := c.String("fingerprint")
							return ultragist.AuthorizedKeys(fingerprint)
						},
					},
				},
			},
			{
				Name:  "db",
				Usage: "interact with the database",
				Subcommands: []*cli.Command{
					{
						Name:  "init",
						Usage: "initialize the database",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "dbpath",
								Usage:    "path to db locally (file:/path) or on gcs (gs://bucket/path/to/db)",
								Required: true,
							},
						},
						Action: func(cCtx *cli.Context) error {
							dbpath := cCtx.String("dbpath")
							ultragist.InitDB(dbpath)
							return nil
						},
					},
					{
						Name:  "test",
						Usage: "try reads and writes to db",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "dbpath",
								Usage:    "path to db locally (file:/path) or on gcs (gs://bucket/path/to/db)",
								Required: true,
							},
						},
						Action: func(cCtx *cli.Context) error {
							dbpath := cCtx.String("dbpath")
							// ultragist.InitDB(gcspath)
							return ultragist.DBTest(dbpath)
						},
					},
					{
						Name:  "export",
						Usage: "export and concat db to local file",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "dbpath",
								Usage:    "path to db on gcs (gs://bucket/path/to/db)",
								Required: true,
							},
							&cli.StringFlag{
								Name:  "dest",
								Usage: "where to write the db",
							},
						},
						Action: func(cCtx *cli.Context) error {
							dbpath := cCtx.String("dbpath")
							dest := cCtx.String("dest")
							return ultragist.DBExport(dbpath, dest)
						},
					},
				},
			},
		},
	}

	err := ultragist.InitSqliteVFS()
	if err != nil {
		log.Fatalf("init sqlite vfs err: %s", err)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
