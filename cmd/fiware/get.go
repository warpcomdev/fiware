package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/iotam"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/orion"
	"github.com/warpcomdev/fiware/internal/perseo"
)

var canGet []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
}

type serializerWithSetup interface {
	fiware.Serializer
	Setup(importer.Writer, map[string]string)
	Begin()
	End()
}

func getConfig(c *cli.Context, store *config.Store) (zero config.Config, h http.Header, err error) {
	if err := store.Read(); err != nil {
		return zero, nil, err
	}
	if store.Current.Name == "" {
		return zero, nil, errors.New("no contexts defined")
	}

	selected := store.Current
	if selected.KeystoneURL == "" || selected.Service == "" || selected.Username == "" {
		return zero, nil, errors.New("current context is not properly configured")
	}
	k, err := keystone.New(selected.KeystoneURL, selected.Username, selected.Service)
	if err != nil {
		return zero, nil, err
	}

	subservice := c.String(subServiceFlag.Name)
	if subservice != "" {
		selected.Subservice = subservice
	}
	if selected.Subservice == "" {
		return zero, nil, errors.New("no subservice selected")
	}

	token := c.String(tokenFlag.Name)
	if token == "" {
		if token = selected.HasToken(); token == "" {
			return zero, nil, errors.New("no token found, please login first")
		}
	}
	header := k.Headers(selected.Subservice, token)
	return selected, header, nil
}

func getResource(c *cli.Context, store *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canGet, ", "))
	}
	selected, header, err := getConfig(c, store)
	if err != nil {
		return err
	}

	output := c.String(outputFlag.Name)
	var outfile *os.File = os.Stdout
	if output != "" {
		outfile, err = os.Create(output)
		if err != nil {
			return err
		}
		defer outfile.Close()
		fmt.Printf("writing output to file %s\n", output)
	}

	vertical := &fiware.Vertical{Subservice: selected.Subservice}
	for _, arg := range c.Args().Slice() {
		switch arg {
		case "devices":
			if err := getDevices(selected, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if err := getServices(selected, header, vertical); err != nil {
				return err
			}
		case "subscriptions":
			fallthrough
		case "subs":
			fallthrough
		case "suscriptions":
			if err := getSuscriptions(selected, header, vertical); err != nil {
				return err
			}
		case "rules":
			if err := getRules(selected, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to get resource %s", arg)
		}
	}

	var lower = strings.ToLower(output)
	var encoder serializerWithSetup
	if output != "" && (strings.HasSuffix(lower, ".py") || strings.HasSuffix(lower, ".star")) {
		ext := filepath.Ext(output)
		encoder = &importer.StarlarkSerializer{
			Name: output[0 : len(output)-len(ext)],
		}
	} else {
		encoder = &importer.JsonnetSerializer{}
	}

	encoder.Setup(outfile, selected.Params)
	encoder.Begin()
	vertical.Serialize(encoder)
	encoder.End()
	if err := encoder.Error(); err != nil {
		return fmt.Errorf("failed to encode: %v", err)
	}
	return nil
}

func getDevices(ctx config.Config, header http.Header, vertical *fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	devices, err := api.Devices(http.DefaultClient, header)
	if err != nil {
		return err
	}
	vertical.Devices = devices
	return nil
}

func getServices(ctx config.Config, header http.Header, vertical *fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	groups, err := api.Services(http.DefaultClient, header)
	if err != nil {
		return err
	}
	vertical.Services = groups
	return nil
}

func getSuscriptions(ctx config.Config, header http.Header, vertical *fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	suscriptions, err := api.Suscriptions(http.DefaultClient, header)
	if err != nil {
		return err
	}
	vertical.Suscriptions = suscriptions
	return nil
}

func getRules(ctx config.Config, header http.Header, vertical *fiware.Vertical) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	rules, err := api.Rules(http.DefaultClient, header)
	if err != nil {
		return err
	}
	vertical.Rules = rules
	return nil
}
