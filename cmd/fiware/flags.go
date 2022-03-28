package main

import "github.com/urfave/cli/v2"

var (
	verticalFlag = &cli.StringFlag{
		Name:        "vertical",
		Aliases:     []string{"v"},
		DefaultText: "vertical",
		Usage:       "vertical name (without '-vertical' suffix)",
		Required:    true,
	}

	subServiceFlag = &cli.StringFlag{
		Name:    "subservice",
		Aliases: []string{"ss"},
		Usage:   "subservice name",
	}

	tokenFlag = &cli.StringFlag{
		Name:        "token",
		Aliases:     []string{"t"},
		Usage:       "authentication token",
		DefaultText: "<empty>",
		EnvVars:     []string{"FIWARE_TOKEN", "X_AUTH_TOKEN"},
	}

	dataFlag = &cli.StringFlag{
		Name:     "data",
		Aliases:  []string{"d"},
		Usage:    "read vertical data from `FILE`",
		Required: true,
	}

	libFlag = &cli.StringFlag{
		Name:    "lib",
		Aliases: []string{"l"},
		Usage:   "load data modules / libs from `DIR`",
	}

	outputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "write output to `FILE`",
	}

	saveFlag = &cli.BoolFlag{
		Name:    "save",
		Aliases: []string{"s"},
		Usage:   "save token to context",
		Hidden:  true,
		Value:   false,
	}
)
