package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

var (
	namespaceFlag = &cli.StringFlag{
		Name:        "namespace",
		Aliases:     []string{"ns", "n"},
		DefaultText: "namespace",
		Usage:       "namespace for the vertical entities",
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
		Aliases:     []string{"urbo-token", "ut", "T"},
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
		Name:    "filter-id",
		Aliases: []string{"fi"},
		Usage:   "Filter by entity ID",
	}

	filterTypeFlag = &cli.StringFlag{
		Name:    "filter-type",
		Aliases: []string{"ft"},
		Usage:   "Filter by entity Type",
	}

	simpleQueryFlag = &cli.StringFlag{
		Name:    "simple-query",
		Aliases: []string{"q"},
		Usage:   "Filter by entity attribs (using NGSIv2 simple query format)",
	}

	outputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "write output to `FILE`",
	}

	verboseFlag = &cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Usage:   "write verbose logging (HTTP requests)",
		Value:   false,
	}

	verbose2Flag = &cli.BoolFlag{
		Name:  "vv",
		Usage: "more verbose logging (HTTP reply headers)",
		Value: false,
	}

	verbose3Flag = &cli.BoolFlag{
		Name:  "vvv",
		Usage: "the most verbose logging (HTTP reply bodies)",
		Value: false,
	}

	// Simplify management of repeated verbosity flags
	verboseFlags = []cli.Flag{verboseFlag, verbose2Flag, verbose3Flag}

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

	portFlag = &cli.IntFlag{
		Name:    "port",
		Aliases: []string{"p"},
		Usage:   "TCP port for server mode",
		Value:   9081,
	}

	maxFlag = &cli.IntFlag{
		Name:    "maximum",
		Aliases: []string{"M", "max"},
		Usage:   "Maximum number of assets to get",
		Value:   0,
	}

	oncePerEntityFlag = &cli.StringFlag{
		Name:    "per-entity",
		Aliases: []string{"E"},
		Usage:   "Run the template once per entity, generating files with this extension",
		Value:   "",
	}

	pepFlag = &cli.BoolFlag{
		Name:  "pep",
		Usage: "Authenticate as PEP user (and get the user_id needed for trusts)",
		Value: false,
	}

	ngsiFlag = &cli.BoolFlag{
		Name:  "ngsi",
		Usage: "When decoding, parse json files as NGSI v2 format entities",
		Value: false,
	}

	assetFlag = &cli.BoolFlag{
		Name:  "asset",
		Usage: "When decoding, parse json files as DEPLOYER top-level assets",
		Value: false,
	}

	timeoutFlag = &cli.IntFlag{
		Name:  "timeout",
		Usage: "Request expiration timeout for requests (in seconds)",
		Value: 15,
	}
)

// verbosity combines info from all verbose flags
func verbosity(c *cli.Context) int {
	verbose := 0
	if c.Bool(verbose3Flag.Name) {
		verbose = 3
	} else if c.Bool(verbose2Flag.Name) {
		verbose = 2
	} else if c.Bool(verboseFlag.Name) {
		// There is some weird bug in urfave/cli 2.24.4 that never counts a flag
		// just once. It is either 0, 2, or more.
		// return c.Count(verboseFlag.Name)
		return 1
	}
	return verbose
}

// verbosity combines info from all verbose flags
func configuredTimeout(c *cli.Context) time.Duration {
	timeout := c.Int(timeoutFlag.Name)
	if timeout < 5 {
		timeout = 15
	}
	if timeout > 1800 {
		timeout = 1800
	}
	return time.Duration(timeout) * time.Second
}
