package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/iotam"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/orion"
	"github.com/warpcomdev/fiware/internal/perseo"
	"github.com/warpcomdev/fiware/internal/serialize"
	"github.com/warpcomdev/fiware/internal/urbo"
)

var canGet []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
	"projects",
	"panels",
	"verticals",
	"entities",
	"regitrations",
}

type serializerWithSetup interface {
	serialize.Serializer
	Setup(serialize.Writer, map[string]string)
	Begin()
	End()
}

func getConfig(c *cli.Context, store *config.Store) (zero config.Config, err error) {
	if err := store.Read(); err != nil {
		return zero, err
	}
	if store.Current.Name == "" {
		return zero, errors.New("no contexts defined")
	}

	selected := store.Current
	if selected.KeystoneURL == "" || selected.Service == "" || selected.Username == "" {
		return zero, errors.New("current context is not properly configured")
	}
	return selected, nil
}

func getKeystoneHeaders(c *cli.Context, selected config.Config) (k *keystone.Keystone, h http.Header, err error) {
	k, err = keystone.New(selected.KeystoneURL, selected.Username, selected.Service)
	if err != nil {
		return nil, nil, err
	}

	subservice := c.String(subServiceFlag.Name)
	if subservice != "" {
		selected.Subservice = subservice
	}
	if selected.Subservice == "" {
		return nil, nil, errors.New("no subservice selected")
	}

	token := c.String(tokenFlag.Name)
	if token == "" {
		if token = selected.HasToken(); token == "" {
			return nil, nil, errors.New("no token found, please login first")
		}
	}
	header := k.Headers(selected.Subservice, token)
	return k, header, nil
}

func getUrboHeaders(c *cli.Context, selected config.Config) (u *urbo.Urbo, h http.Header, err error) {
	u, err = urbo.New(selected.UrboURL, selected.Username, selected.Service, selected.Service)
	if err != nil {
		return nil, nil, err
	}

	subservice := c.String(subServiceFlag.Name)
	if subservice != "" {
		selected.Subservice = subservice
	}
	if selected.Subservice == "" {
		return nil, nil, errors.New("no subservice selected")
	}

	token := c.String(urboTokenFlag.Name)
	if token == "" {
		if token = selected.HasUrboToken(); token == "" {
			return nil, nil, errors.New("no urbo token found, please set `urbo` context var and login first")
		}
	}
	header, err := u.Headers(token)
	return u, header, err
}

func getResource(c *cli.Context, store *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canGet, ", "))
	}
	selected, err := getConfig(c, store)
	if err != nil {
		return err
	}
	output := outputFile(c.String(outputFlag.Name))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()

	vertical := &fiware.Vertical{Subservice: selected.Subservice}
	client := httpClient(c.Bool(verboseFlag.Name))
	for _, arg := range c.Args().Slice() {
		var k *keystone.Keystone
		var u *urbo.Urbo
		var header http.Header
		switch arg {
		case "devices":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := getDevices(selected, client, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := getServices(selected, client, header, vertical); err != nil {
				return err
			}
		case "subscriptions":
			fallthrough
		case "subs":
			fallthrough
		case "suscriptions":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := getSuscriptions(selected, client, header, vertical); err != nil {
				return err
			}
		case "registrations":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := getRegistrations(selected, client, header, vertical); err != nil {
				return err
			}
		case "entities":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			filterId := c.String(filterIdFlag.Name)
			filterType := c.String(filterTypeFlag.Name)
			if err := getEntities(selected, client, header, filterId, filterType, vertical); err != nil {
				return err
			}
		case "rules":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := getRules(selected, client, header, vertical); err != nil {
				return err
			}
		case "projects":
			if k, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := getProjects(selected, client, k, header, vertical); err != nil {
				return err
			}
		case "panels":
			if u, header, err = getUrboHeaders(c, selected); err != nil {
				return err
			}
			if err := getPanels(selected, client, u, header, vertical); err != nil {
				return err
			}
		case "verticals":
			if u, header, err = getUrboHeaders(c, selected); err != nil {
				return err
			}
			if err := getVerticals(selected, client, u, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to get resource %s", arg)
		}
	}

	return output.Encode(outfile, vertical, selected.Params)
}

func getDevices(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	devices, err := api.Devices(c, header)
	if err != nil {
		return err
	}
	vertical.Devices = devices
	return nil
}

func getServices(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	groups, err := api.Services(c, header)
	if err != nil {
		return err
	}
	vertical.Services = groups
	return nil
}

func getSuscriptions(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	suscriptions, err := api.Suscriptions(c, header)
	if err != nil {
		return err
	}
	vertical.Suscriptions = suscriptions
	return nil
}

func getRegistrations(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	registrations, err := api.Registrations(c, header)
	if err != nil {
		return err
	}
	vertical.Registrations = registrations
	return nil
}

func getEntities(ctx config.Config, c keystone.HTTPClient, header http.Header, filterId, filterType string, vertical *fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	types, values, err := api.Entities(c, header, filterId, filterType)
	if err != nil {
		return err
	}
	vertical.EntityTypes = types
	vertical.Entities = values
	return nil
}

func getRules(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Vertical) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	rules, err := api.Rules(c, header)
	if err != nil {
		return err
	}
	namedRules := make(map[string]fiware.Rule, len(rules))
	for _, rule := range rules {
		namedRules[rule.Name] = rule
	}
	vertical.Rules = namedRules
	return nil
}

func getProjects(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Vertical) error {
	projects, err := k.Projects(c, header)
	if err != nil {
		return err
	}
	vertical.Projects = projects
	return nil
}

func getPanels(ctx config.Config, c keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical *fiware.Vertical) error {
	panels, err := u.Panels(c, header)
	if err != nil {
		return err
	}
	vertical.Panels = panels
	return nil
}

func getVerticals(ctx config.Config, c keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical *fiware.Vertical) error {
	verticals, err := u.GetVerticals(c, header)
	if err != nil {
		return err
	}
	vertical.Verticals = verticals
	return nil
}
