package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/decode"
	"github.com/warpcomdev/fiware/internal/template"
)

func firstArg(c *cli.Context, msg string) (string, error) {
	if c.Args().Len() <= 0 {
		return "", errors.New("please provide a name for the new context")
	}
	return c.Args().Get(0), nil
}

func main() {

	dirname, err := os.UserConfigDir()
	if err != nil {
		log.Print("Failed to locate user config dir, defaulting to /tmp")
		dirname = "/tmp"
	}
	defaultStore := path.Join(dirname, "fiware.json")
	currentStore := &config.Store{}

	// Autocomplete enumera los contextos disponibles para autocompletado
	autocomplete := func(c *cli.Context) {
		if c.NArg() > 0 {
			return
		}
		if currentStore.Path != "" {
			configs, err := currentStore.List()
			if err == nil {
				fmt.Println(strings.Join(configs, "\n"))
			}
		}
	}

	app := &cli.App{

		Description: "Manage fiware verticals and environments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "context",
				Aliases:     []string{"c"},
				Usage:       "Path to the context configuration file",
				Value:       defaultStore,
				DefaultText: "${XDG_CONFIG_DIR}/fiware.json",
				EnvVars:     []string{"FIWARE_CONTEXT"},
			},
		},
		Before: func(c *cli.Context) error {
			currentStore.Path = c.String("context")
			return nil
		},
		EnableBashCompletion: true,

		Commands: []*cli.Command{

			{
				Name:     "decode",
				Category: "template",
				Usage:    "decode NGSI README.md or CSV file",
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						return errors.New("please provide the path to NGSI README file")
					}
					return decode.Decode(
						c.String("output"),
						c.String("vertical"),
						c.String("subservice"),
						c.Args().Get(0),
					)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "write output to `FILE`",
					},
					&cli.StringFlag{
						Name:        "vertical",
						Aliases:     []string{"v"},
						DefaultText: "vertical",
						Usage:       "vertical name (without '-vertical' suffix)",
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "subservice",
						Aliases:     []string{"ss"},
						DefaultText: "subservice",
						Usage:       "subservice name (without '/' prefix)",
						Required:    true,
					},
				},
			},

			{
				Name:     "template",
				Category: "template",
				Usage:    "template for vertical data",
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						return errors.New("please provide the path to NGSI README file")
					}
					if err := currentStore.Read(); err != nil {
						return err
					}
					var params map[string]string
					selected := currentStore.Current
					if selected.Name != "" {
						if len(selected.Params) > 0 {
							params = selected.Params
						}
					}
					return render(c, params)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "read vertical data from `FILE`",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "write template output to `FILE`",
					},
				},
				BashComplete: func(c *cli.Context) {
					if c.NArg() <= 0 {
						builtins, err := template.Builtins()
						if err == nil {
							fmt.Println(strings.Join(builtins, "\n"))
						}
					}
				},
			},

			{
				Name:     "login",
				Category: "platform",
				Aliases:  []string{"auth"},
				Usage:    "Login into keystone",
				Action: func(c *cli.Context) error {
					return auth(c, currentStore)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "save",
						Aliases: []string{"s"},
						Usage:   "save token to context",
						Hidden:  true,
						Value:   false,
					},
				},
			},

			{
				Name:     "get",
				Category: "platform",
				Usage:    fmt.Sprintf("Get some resource (%s)", strings.Join(canGet, ", ")),
				BashComplete: func(c *cli.Context) {
					fmt.Println(strings.Join(canGet, "\n"))
				},
				Action: func(c *cli.Context) error {
					return getResource(c, currentStore)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "token",
						Aliases:     []string{"t"},
						Usage:       "authentication token",
						DefaultText: "<empty>",
						EnvVars:     []string{"FIWARE_TOKEN", "X_AUTH_TOKEN"},
					},
					&cli.StringFlag{
						Name:    "subservice",
						Aliases: []string{"ss"},
						Usage:   "subservice name",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Write output to `FILE`",
					},
				},
			},

			{
				Name:     "post",
				Category: "platform",
				Usage:    fmt.Sprintf("Post some resource (%s)", strings.Join(canPost, ", ")),
				BashComplete: func(c *cli.Context) {
					fmt.Println(strings.Join(canPost, "\n"))
				},
				Action: func(c *cli.Context) error {
					return postResource(c, currentStore)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "token",
						Aliases:     []string{"t"},
						Usage:       "authentication token",
						DefaultText: "<empty>",
						EnvVars:     []string{"FIWARE_TOKEN", "X_AUTH_TOKEN"},
					},
					&cli.StringFlag{
						Name:    "subservice",
						Aliases: []string{"ss"},
						Usage:   "subservice name",
					},
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "Read vertical data from `FILE`",
						Required: true,
					},
				},
			},

			{
				Name:     "delete",
				Category: "platform",
				Usage:    fmt.Sprintf("Delete some resource (%s)", strings.Join(canDelete, ", ")),
				BashComplete: func(c *cli.Context) {
					fmt.Println(strings.Join(canPost, "\n"))
				},
				Action: func(c *cli.Context) error {
					return deleteResource(c, currentStore)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "token",
						Aliases:     []string{"t"},
						Usage:       "authentication token",
						DefaultText: "<empty>",
						EnvVars:     []string{"FIWARE_TOKEN", "X_AUTH_TOKEN"},
					},
					&cli.StringFlag{
						Name:    "subservice",
						Aliases: []string{"ss"},
						Usage:   "subservice name",
					},
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "Read vertical data from `FILE`",
						Required: true,
					},
				},
			},

			{
				Name:     "context",
				Category: "config",
				Aliases:  []string{"ctx"},
				Usage:    "Manage contexts",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create a new context",
						Action: func(c *cli.Context) error {
							return createContext(currentStore, c)
						},
					},
					{
						Name:    "delete",
						Aliases: []string{"rm"},
						Usage:   "Delete a context",
						Action: func(c *cli.Context) error {
							return deleteContext(currentStore, c)
						},
						BashComplete: autocomplete,
					},
					{
						Name:    "list",
						Aliases: []string{"ls"},
						Usage:   "List all contexts",
						Action: func(c *cli.Context) error {
							return listContext(currentStore, c)
						},
					},
					{
						Name:  "use",
						Usage: "Use a context",
						Action: func(c *cli.Context) error {
							return useContext(currentStore, c)
						},
						BashComplete: autocomplete,
					},
					{
						Name:    "info",
						Usage:   "Show context configuration",
						Aliases: []string{"show"},
						Action: func(c *cli.Context) error {
							return infoContext(currentStore, c)
						},
						BashComplete: autocomplete,
					},
					{
						Name:  "dup",
						Usage: "Duplicate the current context",
						Action: func(c *cli.Context) error {
							return dupContext(currentStore, c)
						},
					},
					{
						Name:  "set",
						Usage: "Set a context variable",
						Action: func(c *cli.Context) error {
							nargs := c.NArg()
							if nargs > 0 && (nargs%2 == 0) {
								return setContext(currentStore, c, c.Args().Slice())
							}
							return errors.New("please introduce variable - value pairs")
						},
						BashComplete: func(c *cli.Context) {
							nargs := c.NArg()
							if nargs%2 == 1 {
								return
							}
							fmt.Println(strings.Join(currentStore.CanConfig(), "\n"))
						},
					},

					{
						Name:  "params",
						Usage: "Set a template parameter",
						Action: func(c *cli.Context) error {
							nargs := c.NArg()
							if nargs > 0 && (nargs%2 == 0) {
								return setParamsContext(currentStore, c, c.Args().Slice())
							}
							return errors.New("please introduce variable - value pairs")
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
