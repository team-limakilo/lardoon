package main

import (
	"log"
	"os"
	"time"

	"github.com/b1naryth1ef/lardoon"
	"github.com/urfave/cli/v2"
)

func doServe(c *cli.Context) error {
	err := lardoon.InitDatabase(c.Path("db"))
	if err != nil {
		return err
	}

	var server lardoon.HTTPServer
	return server.Run(c.String("bind"))
}

func doImport(c *cli.Context) error {
	err := lardoon.InitDatabase(c.Path("db"))
	if err != nil {
		return err
	}

	return lardoon.ImportPath(c.Path("import-path"))
}

func doPrune(c *cli.Context) error {
	err := lardoon.InitDatabase(c.Path("db"))
	if err != nil {
		return err
	}

	return lardoon.PruneReplays(!c.Bool("no-dry-run"))
}

func doDaemon(c *cli.Context) error {
	err := lardoon.InitDatabase(c.Path("db"))
	if err != nil {
		return err
	}

	for {
		err = lardoon.ImportPath(c.Path("import-path"))
		if err != nil {
			log.Printf("[daemon] Import error: %v", err)
		}

		err = lardoon.PruneReplays(false)
		if err != nil {
			log.Printf("[daemon] Prune error: %v", err)
		}

		time.Sleep(time.Duration(c.Int64("time-period")) * time.Second)
	}
}

func main() {
	app := &cli.App{
		Name:  "lardoon",
		Usage: "ACMI repository and trimming service",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "db",
				Usage:   "path to sqlite3 database file",
				Value:   "lardoon.db",
				EnvVars: []string{"LARDOON_DB_PATH"},
			},
		},
		Action: func(ctx *cli.Context) error {
			cli.ShowAppHelp(ctx)
			return cli.Exit("", 2)
		},
		Commands: []*cli.Command{
			{
				Name:   "prune",
				Action: doPrune,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "no-dry-run",
						Usage: "during a dry-run no data will be mutated",
						Value: false,
					},
				},
			},
			{
				Name:   "import",
				Action: doImport,
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "import-path",
						Usage:    "directory or replay path to import",
						Required: true,
						Aliases:  []string{"p"},
					},
				},
			},
			{
				Name:   "serve",
				Action: doServe,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "bind",
						Usage: "hostname/port to bind the server on",
						Value: "localhost:3883",
					},
				},
			},
			{
				Name:        "daemon",
				Action:      doDaemon,
				Description: "Runs import and prune commands continuously",
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:     "import-path",
						Usage:    "directory or replay path to import",
						Required: true,
						Aliases:  []string{"p"},
					},
					&cli.Int64Flag{
						Name:        "time-period",
						Usage:       "time to wait between runs",
						Required:    true,
						DefaultText: "unset",
						Aliases:     []string{"t"},
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
