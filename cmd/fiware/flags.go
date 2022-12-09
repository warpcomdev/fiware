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

	urboTokenFlag = &cli.StringFlag{
		Name:        "urbotoken",
		Aliases:     []string{"urboToken", "ut", "T"},
		Usage:       "Urbo authentication token",
		DefaultText: "<empty>",
		EnvVars:     []string{"URBO_TOKEN", "AUTHORIZATION_TOKEN"},
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

	filterIdFlag = &cli.StringFlag{
		Name:    "filterId",
		Aliases: []string{"fi"},
		Usage:   "Filter by entity ID",
	}

	filterTypeFlag = &cli.StringFlag{
		Name:    "filterType",
		Aliases: []string{"ft"},
		Usage:   "Filter by entity Type",
	}

	simpleQueryFlag = &cli.StringFlag{
		Name:    "simpleQuery",
		Aliases: []string{"q"},
		Usage:   "Filter by entity ID",
	}

	outputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "write output to `FILE`",
	}

	verboseFlag = &cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"V"},
		Usage:   "write verbose logging",
		Value:   false,
	}

	allFlag = &cli.BoolFlag{
		Name:  "all",
		Usage: "Download all verticals or subservices",
		Value: false,
	}

	outdirFlag = &cli.StringFlag{
		Name:     "outdir",
		Aliases:  []string{"o", "O"},
		Usage:    "write output to `DIR`",
		Required: true,
	}

	saveFlag = &cli.BoolFlag{
		Name:    "save",
		Aliases: []string{"s"},
		Usage:   "save token to context",
		Hidden:  true,
		Value:   false,
	}

	relaxedFlag = &cli.BoolFlag{
		Name:  "relaxed",
		Usage: "Do not validate data schema",
		Value: false,
	}

	useExactIdFlag = &cli.BoolFlag{
		Name:    "exact",
		Aliases: []string{"x"},
		Usage:   "Match subscriptions by exact ID instead of description",
		Value:   false,
	}

	selectedContextFlag = &cli.StringFlag{
		Name:    "context",
		Aliases: []string{"c"},
		Usage:   "Use an alternative context (instead of the one in use)",
	}
)
