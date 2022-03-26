package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/iotam"
	"github.com/warpcomdev/fiware/internal/orion"
	"github.com/warpcomdev/fiware/internal/perseo"
	"github.com/warpcomdev/fiware/internal/template"
)

var canDelete []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
}

func deleteResource(c *cli.Context, store *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canPost, ", "))
	}

	selected, header, err := getConfig(c, store)
	if err != nil {
		return err
	}

	var vertical fiware.Vertical
	if err := template.Load(c.String("data"), &vertical); err != nil {
		return err
	}

	for _, arg := range c.Args().Slice() {
		switch arg {
		case "devices":
			if err := deleteDevices(selected, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if err := deleteServices(selected, header, vertical); err != nil {
				return err
			}
		case "subscriptions":
			fallthrough
		case "subs":
			fallthrough
		case "suscriptions":
			if err := deleteSuscriptions(selected, header, vertical); err != nil {
				return err
			}
		case "rules":
			if err := deleteRules(selected, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to delete resource %s", arg)
		}
	}
	return nil
}

func deleteDevices(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("DELETing devices with IDs", vertical.Devices,
		func(g fiware.Device) string { return g.DeviceId },
	)
	return api.DeleteDevices(http.DefaultClient, header, vertical.Devices)
}

func deleteServices(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("DELETing groups with API Keys", vertical.Services,
		func(g fiware.Service) string { return g.APIKey },
	)
	return api.DeleteServices(http.DefaultClient, header, vertical.Services)
}

func deleteSuscriptions(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	listMessage("DELETing suscriptions with descriptions", vertical.Suscriptions,
		func(g fiware.Suscription) string { return g.Description },
	)
	return api.DeleteSuscriptions(http.DefaultClient, header, vertical.Suscriptions)
}

func deleteRules(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	listMessage("DELETing rules with names", vertical.Rules,
		func(g fiware.Rule) string { return g.Name },
	)
	return api.DeleteRules(http.DefaultClient, header, vertical.Rules)
}
