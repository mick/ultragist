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
				Name:  "init-db",
				Usage: "initialize the database",
				Action: func(cCtx *cli.Context) error {
					ultragist.InitDB()
					return nil
				},
			},
			{
				Name:  "db-test",
				Usage: "try reads and writes to db",
				Action: func(cCtx *cli.Context) error {
					// ultragist.InitDB()
					return ultragist.DBTest()
				},
			},
			{
				Name:  "db-export",
				Usage: "export and concat db to local file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "gcspath",
						Usage:    "path to db in gcs",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "dest",
						Usage: "where to write the db",
					},
				},
				Action: func(cCtx *cli.Context) error {
					gcspath := cCtx.String("gcspath")
					dest := cCtx.String("dest")
					return ultragist.DBExport(gcspath, dest)
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
