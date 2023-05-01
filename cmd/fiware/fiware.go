package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/decode"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/snapshots"
	"github.com/warpcomdev/fiware/internal/template"
	"github.com/warpcomdev/fiware/internal/urbo"
)

// Autocompleter builds an autocomplete function for projects and (optionally) subservices
func autocompleter(currentStore *config.Store, subservices bool) func(c *cli.Context) {
	return func(c *cli.Context) {
		if c.NArg() > 1 {
			return
		}
		currentStore.Path = c.String("context")
		if c.NArg() < 1 {
			names, err := currentStore.List(true)
			if err != nil {
				log.Printf("Error listing contexts: %s", err)
				return
			}
			fmt.Println(strings.Join(names, "\n"))
			return
		}
		// If c.NArgs() == 1, some context is already selected.
		// Try to autocomplete with subservices, if the command
		// supports it.
		if subservices {
			subserviceComplete(currentStore, c.Args().Get(0))
		}
	}
}

// SubserviceComplete prints a list of subservices for the given context
func subserviceComplete(currentStore *config.Store, contextName string) {
	selected, err := currentStore.Info(contextName)
	if err != nil {
		log.Printf("Error getting info for config: %s", err)
		return
	}
	fmt.Println(strings.Join(selected.ProjectCache, "\n"))
}

func main() {

	dirname, err := os.UserConfigDir()
	if err != nil {
		log.Print("Failed to locate user config dir, defaulting to /tmp")
		dirname = "/tmp"
	}
	defaultStore := path.Join(dirname, "fiware.json")
	currentStore := &config.Store{}

	// Backoff policy
	backoff := keystone.ExponentialBackoff{
		MaxRetries:   3,
		InitialDelay: 2 * time.Second,
		DelayFactor:  2,
		MaxDelay:     10 * time.Second,
	}

	app := &cli.App{

		Name:        "FIWARE CLI client",
		Usage:       "manage fiware environments",
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
				Usage:    "decode NGSI README.md, CSV file or builder json model",
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						return errors.New("please provide the path to NGSI README file")
					}
					format := ""
					if c.Bool(ngsiFlag.Name) {
						format = decode.FORMAT_NGSI
					}
					if c.Bool(assetFlag.Name) {
						format = decode.FORMAT_ASSET
					}
					return decode.Decode(
						c.String(outputFlag.Name),
						c.String(namespaceFlag.Name),
						c.String(subServiceFlag.Name),
						c.Args().Slice(),
						format,
					)
				},
				Flags: []cli.Flag{
					outputFlag,
					namespaceFlag,
					subServiceFlag,
					ngsiFlag,
					assetFlag,
				},
			},

			{
				Name:     "export",
				Category: "template",
				Usage:    "Read datafile and export with context params",
				Action: func(c *cli.Context) error {
					if err := currentStore.Read(""); err != nil {
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
					if err := currentStore.Read(""); err != nil {
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
					oncePerEntityFlag,
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
					if c.Bool(pepFlag.Name) {
						return authAsPep(c, currentStore, backoff)
					}
					return auth(c, currentStore, backoff)
				},
				Flags: append([]cli.Flag{
					selectedContextFlag,
					pepFlag,
					saveFlag,
					timeoutFlag,
				}, verboseFlags...),
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
				Flags: append([]cli.Flag{
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					outputFlag,
					filterIdFlag,
					filterTypeFlag,
					simpleQueryFlag,
					maxFlag,
					timeoutFlag,
				}, verboseFlags...),
			},

			{
				Name:     "download",
				Aliases:  []string{"down", "dld"},
				Category: "platform",
				Usage:    fmt.Sprintf("Download vertical or subservice"),
				Subcommands: []*cli.Command{

					&(cli.Command{
						Name:    "verticals",
						Aliases: []string{"vertical", "v"},
						Usage:   fmt.Sprintf("Download panels from vertical(s)"),
						BashComplete: func(c *cli.Context) {
							v, err := newVerticalDownloader(c, currentStore)
							if err != nil {
								fmt.Println("<log in first>")
							} else {
								fmt.Println(strings.Join(v.List(), "\n"))
							}
						},
						Action: func(c *cli.Context) error {
							v, err := newVerticalDownloader(c, currentStore)
							if err != nil {
								return err
							}
							return v.Download(c, currentStore)
						},
						Flags: append([]cli.Flag{
							outdirFlag,
							urboTokenFlag,
							allFlag,
							maxFlag,
						}, verboseFlags...),
					}),

					&(cli.Command{
						Name:    "subservices",
						Aliases: []string{"subservice", "ss", "s"},
						Usage:   fmt.Sprintf("Download resources from subservice(s)"),
						BashComplete: func(c *cli.Context) {
							v, err := newProjectDownloader(c, currentStore)
							if err != nil {
								fmt.Println("<log in first>")
							} else {
								fmt.Println(strings.Join(v.List(), "\n"))
							}
						},
						Action: func(c *cli.Context) error {
							v, err := newProjectDownloader(c, currentStore)
							if err != nil {
								return err
							}
							return v.Download(c, currentStore)
						},
						Flags: append([]cli.Flag{
							outdirFlag,
							tokenFlag,
							allFlag,
							timeoutFlag,
						}, verboseFlags...),
					}),
				},
			},

			{
				Name:    "upload",
				Aliases: []string{"up"},
				Usage:   fmt.Sprintf("Upload panels to urbo"),
				Action: func(c *cli.Context) error {
					return uploadResource(c, currentStore)
				},
				Flags: append([]cli.Flag{
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					timeoutFlag,
				}, verboseFlags...),
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
				Flags: append([]cli.Flag{
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					dataFlag,
					libFlag,
					useExactIdFlag,
					filterTypeFlag,
					timeoutFlag,
				}, verboseFlags...),
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
				Flags: append([]cli.Flag{
					tokenFlag,
					urboTokenFlag,
					subServiceFlag,
					dataFlag,
					libFlag,
					useExactIdFlag,
					filterTypeFlag,
					timeoutFlag,
				}, verboseFlags...),
			},

			{
				Name:     "context",
				Category: "config",
				Aliases:  []string{"ctx"},
				Usage:    "Manage contexts",
				Action: func(c *cli.Context) error {
					// Default action just prints selected context,
					// to simplify spotting which context is selected
					if err := currentStore.Use(""); err != nil {
						return err
					}
					summaryContext(currentStore.Current)
					return nil
				},
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
						BashComplete: autocompleter(currentStore, false),
					},
					{
						Name:    "list",
						Aliases: []string{"ls"},
						Usage:   "List all contexts",
						Action: func(c *cli.Context) error {
							return listContext(currentStore, c, true)
						},
					},
					{
						Name:  "use",
						Usage: "Use a context",
						Action: func(c *cli.Context) error {
							return useContext(currentStore, c)
						},
						BashComplete: autocompleter(currentStore, true),
					},
					{
						Name:    "info",
						Usage:   "Show context configuration",
						Aliases: []string{"show"},
						Action: func(c *cli.Context) error {
							return infoContext(currentStore, c)
						},
						BashComplete: autocompleter(currentStore, false),
					},
					{
						Name:  "dup",
						Usage: "Duplicate the current context",
						Action: func(c *cli.Context) error {
							return dupContext(currentStore, c)
						},
					},
					{
						Name:  "env",
						Usage: "Dump as an urbo-deployer Environment",
						Action: func(c *cli.Context) error {
							return envContext(currentStore, c)
						},
					},
					{
						Name:  "set",
						Usage: "Set a context variable",
						Action: func(c *cli.Context) error {
							nargs := c.NArg()
							if nargs > 0 && (nargs%2 == 0) {
								return setContext(currentStore, c, "", c.Args().Slice())
							}
							return errors.New("please introduce variable - value pairs")
						},
						BashComplete: func(c *cli.Context) {
							nargs := c.NArg()
							if nargs%2 == 1 {
								// subservice can also be autocompleted
								lastArg := c.Args().Get(nargs - 1)
								if lastArg == "subservice" || lastArg == "ss" {
									// must prep currentStore for it to work
									// from autocomplete functions
									currentStore.Path = c.String("context")
									subserviceComplete(currentStore, "")
								}
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
								return setParamsContext(currentStore, c, "", c.Args().Slice())
							}
							return errors.New("please introduce variable - value pairs")
						},
					},
				},
			},

			{
				Name:     "serve",
				Category: "platform",
				Usage:    fmt.Sprintf("Turn on http server"),
				Action: func(c *cli.Context) error {
					client := httpClient(0, 15*time.Second)
					mux := &http.ServeMux{}
					mux.Handle("/api/auth", cors(authServe(client, currentStore, backoff)))
					mux.Handle("/api/contexts/", cors(http.StripPrefix("/api/contexts", currentStore.Server())))
					mux.Handle("/api/snaps/", cors(http.StripPrefix("/api/snaps", snapshots.Serve(client, currentStore))))
					mux.Handle("/api/urbo/", cors(http.StripPrefix("/api/urbo", urbo.Serve(client, currentStore))))
					if c.NArg() > 0 {
						mux.Handle("/legacy", http.HandlerFunc(onRenderRequest))
						serveFS := os.DirFS(c.Args().First())
						mux.Handle("/", http.FileServer(http.FS(serveFS)))
					} else {
						mux.Handle("/", http.HandlerFunc(onRenderRequest))
					}
					port := c.Int(portFlag.Name)
					fmt.Printf("Listening at port %d\n", port)
					addr := fmt.Sprintf(":%d", port)
					http.ListenAndServe(addr, mux)
					return nil
				},
				Flags: []cli.Flag{
					portFlag,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
