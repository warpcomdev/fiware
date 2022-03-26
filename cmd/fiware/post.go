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

var canPost []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
}

func postResource(c *cli.Context, config *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canPost, ", "))
	}

	selected, header, err := getConfig(c, config)
	if err != nil {
		return err
	}

	var vertical fiware.Vertical
	if err := template.Load(c.String("data"), selected.Params, &vertical); err != nil {
		return err
	}

	for _, arg := range c.Args().Slice() {
		switch arg {
		case "devices":
			if err := postDevices(selected, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if err := postServices(selected, header, vertical); err != nil {
				return err
			}
		case "subscriptions":
			fallthrough
		case "subs":
			fallthrough
		case "suscriptions":
			if err := postSuscriptions(selected, header, vertical); err != nil {
				return err
			}
		case "rules":
			if err := postRules(selected, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to post resource %s", arg)
		}
	}
	return nil
}

func postDevices(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing devices with IDs", vertical.Devices,
		func(g fiware.Device) string { return g.DeviceId },
	)
	return api.PostDevices(http.DefaultClient, header, vertical.Devices)
}

func postServices(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing groups with API Keys", vertical.Services,
		func(g fiware.Service) string { return g.APIKey },
	)
	return api.PostServices(http.DefaultClient, header, vertical.Services)
}

func postSuscriptions(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	listMessage("POSTing suscriptions with descriptions", vertical.Suscriptions,
		func(g fiware.Suscription) string { return g.Description },
	)
	return api.PostSuscriptions(http.DefaultClient, header, vertical.Suscriptions)
}

func postRules(ctx config.Config, header http.Header, vertical fiware.Vertical) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	listMessage("POSTing rules with names", vertical.Rules,
		func(g fiware.Rule) string { return g.Name },
	)
	return api.PostRules(http.DefaultClient, header, vertical.Rules)
}

func listMessage[T any](msg string, items []T, label func(T) string) {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, label(item))
	}
	fmt.Printf("%s '%s'\n", msg, strings.Join(labels, "','"))
}
