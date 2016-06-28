package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	enableDebugTransport()

	if err := run(); err != nil {
		ExitOnError(err)
	}
}

func run() error {
	logrus.SetLevel(logrus.DebugLevel)

	app := cli.NewApp()
	app.Name = "ConoHa Net"
	app.Usage = "Security group management tool for ConoHa"
	app.Version = "0.1"

	app.Commands = commands
	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}

func ExitOnError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}
