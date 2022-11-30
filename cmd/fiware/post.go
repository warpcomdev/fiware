package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/iotam"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/orion"
	"github.com/warpcomdev/fiware/internal/perseo"
	"github.com/warpcomdev/fiware/internal/urbo"
)

var canPost []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
	"entities",
	"verticals",
}

func postResource(c *cli.Context, config *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canPost, ", "))
	}

	selected, err := getConfig(c, config)
	if err != nil {
		return err
	}

	datapath, libpath := c.String(dataFlag.Name), c.String(libFlag.Name)
	var vertical fiware.Vertical
	if err := importer.Load(datapath, selected.Params, &vertical, libpath); err != nil {
		return err
	}

	client := httpClient(c.Bool(verboseFlag.Name))
	for _, arg := range c.Args().Slice() {
		var u *urbo.Urbo
		var header http.Header
		switch arg {
		case "devices":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := postDevices(selected, client, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := postServices(selected, client, header, vertical); err != nil {
				return err
			}
		case "subscriptions":
			fallthrough
		case "subs":
			fallthrough
		case "suscriptions":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := postSuscriptions(selected, client, header, vertical); err != nil {
				return err
			}
		case "rules":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := postRules(selected, client, header, vertical); err != nil {
				return err
			}
		case "entities":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := postEntities(selected, client, header, vertical); err != nil {
				return err
			}
		case "verticals":
			if u, header, err = getUrboHeaders(c, selected); err != nil {
				return err
			}
			if err := postVerticals(selected, client, u, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to post resource %s", arg)
		}
	}
	return nil
}

func postDevices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing devices with IDs", vertical.Devices,
		func(g fiware.Device) string { return g.DeviceId },
	)
	return api.PostDevices(client, header, vertical.Devices)
}

func postServices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing groups with API Keys", vertical.Services,
		func(g fiware.Service) string { return g.APIKey },
	)
	return api.PostServices(client, header, vertical.Services)
}

func postSuscriptions(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	listMessage("POSTing suscriptions with descriptions", vertical.Suscriptions,
		func(g fiware.Suscription) string { return g.Description },
	)
	return api.PostSuscriptions(client, header, vertical.Suscriptions)
}

func postRules(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	ruleNames, err := vertical.RuleNames()
	if err != nil {
		return err
	}
	listMessage("POSTing rules with names", ruleNames,
		func(g string) string { return g },
	)
	return api.PostRules(client, header, vertical.RuleValues())
}

func postEntities(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	merged := orion.Merge(vertical.EntityTypes, vertical.Entities)
	listMessage("POSTing entities with names", merged,
		func(g orion.Entity) string { return fmt.Sprintf("%s/%s", g.Type(), g.ID()) },
	)
	return api.UpdateEntities(client, header, merged)
}

func postVerticals(ctx config.Config, client keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical fiware.Vertical) error {
	verticalList := make([]fiware.UrboVertical, 0, len(vertical.Verticals))
	for _, v := range vertical.Verticals {
		verticalList = append(verticalList, v)
	}
	listMessage("POSTing verticals with slugs", verticalList,
		func(g fiware.UrboVertical) string { return g.Slug },
	)
	return u.PostVerticals(client, header, vertical.Verticals)
}

func listMessage[T any](msg string, items []T, label func(T) string) {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, label(item))
	}
	fmt.Printf("%s '%s'\n", msg, strings.Join(labels, "','"))
}
