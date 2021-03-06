package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/decode"
	"github.com/warpcomdev/fiware/internal/template"
)

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
				Aliases:  []string{"import"},
				Category: "template",
				Usage:    "decode NGSI README.md or CSV file",
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						return errors.New("please provide the path to NGSI README file")
					}
					return decode.Decode(
						c.String(outputFlag.Name),
						c.String(verticalFlag.Name),
						c.String(subServiceFlag.Name),
						c.Args().Slice(),
					)
				},
				Flags: []cli.Flag{
					outputFlag,
					verticalFlag,
					subServiceFlag,
				},
			},

			{
				Name:     "export",
				Category: "template",
				Usage:    "Read datafile and export with context params",
				Action: func(c *cli.Context) error {
					if err := currentStore.Read(); err != nil {
						return err
					}
					if currentStore.Current.Name == "" {
						return errors.New("no contexts defined")
					}
					selected := currentStore.Current
					return export(c, selected.Params)
				},
				Flags: []cli.Flag{
					dataFlag,
					libFlag,
					outputFlag,
				},
			},

			{
				Name:     "template",
				Category: "template",
				Usage:    "template for vertical data",
				UsageText: func() string {
					msg := append([]string{}, "provide the path to the template file, or the name of a builtin one:\n")
					if builtins, err := template.Builtins(); err == nil {
						// Ordeno los builtins por nombre, con los ficheros primero
						less := func(i, j int) bool {
							iFile := strings.HasSuffix(builtins[i], ".tmpl")
							jFile := strings.HasSuffix(builtins[j], ".tmpl")
							if iFile && !jFile {
								return true
							}
							if jFile && !iFile {
								return false
							}
							return strings.Compare(builtins[i], builtins[j]) < 0
						}
						sort.Slice(builtins, less)
						for _, builtin := range builtins {
							msg = append(msg, fmt.Sprintf("- %s", builtin))
						}
					}
					return strings.Join(msg, "\n")
				}(),
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						return errors.New("please provide the path to the template file")
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
					dataFlag,
					libFlag,
					outputFlag,
					relaxedFlag,
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
					saveFlag,
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
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					outputFlag,
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
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					dataFlag,
					libFlag,
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
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					dataFlag,
					libFlag,
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
