package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	app := cli.NewApp()
	app.Name = "ConoHa Net"
	app.Usage = "Security group management tool for ConoHa"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug,d",
			Usage: "print debug informations.",
		},
		cli.StringFlag{
			Name:  "output,o",
			Usage: `specify output type. must be either "text" or "json". default is "json". `,
			Value: "text",
		},
	}

	// debug
	app.Before = func(c *cli.Context) error {
		if c.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
			enableDebugTransport()
		}
		return nil
	}

	app.Commands = commands
	return app.Run(os.Args)
}
