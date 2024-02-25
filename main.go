package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const usage = `mydocker is a simple container runtime implementation.
			   The purpose of this project is to learn how docker works and how to write a docker by ourselves
			   Enjoy it, just for fun.`

func main() {
	app := cli.NewApp()
	app.Name = "Mydocker"
	app.Usage = usage

	// init command params
	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		commitCommand,
		listCommand,
		logCommand,
		execCommand,
		stopCommand,
		removeCommand,
		networkCommand,
	}

	// init logrus configs
	app.Before = func(ctx *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// urfaveCli-demo
func urfaveCli() {
	app := cli.NewApp()
	app.Name = "demo"
	app.Usage = usage
	// global config
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "lang, l",
			Value: "english",
			Usage: "Language for the greeting",
		},
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load configuration from `FILE`",
		},
	}
	// support command lists, for example go run main.go -c xxx -a xxx
	app.Commands = []cli.Command{
		{
			Name:    "complete",
			Aliases: []string{"c"},
			Usage:   "complete one task",
			Action: func(c *cli.Context) error {
				log.Println("run command and complete")
				for i, v := range c.Args() {
					log.Printf("args i:%v v:%v\n", i, v)
				}
				return nil
			},
		},
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "add one task",
			Flags: []cli.Flag{cli.Int64Flag{
				Name:  "priority",
				Value: 1,
				Usage: "priority for current task",
			}},
			Action: func(c *cli.Context) error {
				log.Println("run command add")
				for i, v := range c.Args() {
					log.Printf("args i:%v v:%v\n", i, v)
				}
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
