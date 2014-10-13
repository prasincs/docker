package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var (
	logger = logrus.New()
)

func main() {
	app := cli.NewApp()
	app.Name = "citadel"
	app.Usage = "docker clusters"
	app.Flags = []cli.Flag{
		cli.StringSliceFlag{Name: "etcd", Value: &cli.StringSlice{}, Usage: "list of etcd machines"},
		cli.BoolFlag{Name: "debug", Usage: "enable debug output"},
	}

	app.Commands = []cli.Command{
		serveCommand,
	}

	app.Before = func(context *cli.Context) error {
		if context.GlobalBool("debug") {
			logger.Level = logrus.DebugLevel
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
