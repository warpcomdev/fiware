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
	manifest, err := importer.Load(datapath, selected.Params, libpath)
	if err != nil {
		return err
	}

	useDescription := !c.Bool(useExactIdFlag.Name)
	client := httpClient(c.Bool(verboseFlag.Name))
	for _, arg := range c.Args().Slice() {
		var u *urbo.Urbo
		var header http.Header
		switch arg {
		case "devices":
			if _, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postDevices(selected, client, header, manifest); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if _, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postServices(selected, client, header, manifest); err != nil {
				return err
			}
		case "subscriptions":
			fallthrough
		case "subs":
			fallthrough
		case "suscriptions":
			if _, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postSuscriptions(selected, client, header, manifest, useDescription); err != nil {
				return err
			}
		case "rules":
			if _, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postRules(selected, client, header, manifest); err != nil {
				return err
			}
		case "entities":
			if _, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postEntities(selected, client, header, manifest); err != nil {
				return err
			}
		case "verticals":
			if u, header, err = getUrboHeaders(c, &selected); err != nil {
				return err
			}
			if err := postVerticals(selected, client, u, header, manifest); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to post resource %s", arg)
		}
	}
	return nil
}

func postDevices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Manifest) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing devices with IDs", vertical.Devices,
		func(g fiware.Device) string { return g.DeviceId },
	)
	return api.PostDevices(client, header, vertical.Devices)
}

func postServices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Manifest) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing groups with API Keys", vertical.Services,
		func(g fiware.Service) string { return g.APIKey },
	)
	return api.PostServices(client, header, vertical.Services)
}

func postSuscriptions(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Manifest, useDescription bool) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	dictMessage("POSTing suscriptions with descriptions", vertical.Subscriptions,
		func(k string, v fiware.Subscription) string { return v.Description },
	)
	return api.PostSuscriptions(client, header, fiware.ValuesOf(vertical.Subscriptions), vertical.Environment.NotificationEndpoints, useDescription)
}

func postRules(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Manifest) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	dictMessage("POSTing rules with names", vertical.Rules,
		func(k string, v fiware.Rule) string {
			if v.Name != "" {
				return v.Name
			}
			return k
		},
	)
	return api.PostRules(client, header, fiware.ValuesOf(vertical.Rules))
}

func postEntities(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Manifest) error {
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

func postVerticals(ctx config.Config, client keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical fiware.Manifest) error {
	dictMessage("POSTing verticals with slugs", vertical.Verticals,
		func(k string, v fiware.Vertical) string { return v.Slug },
	)
	return u.PostVerticals(client, header, vertical.Verticals)
}

func dictMessage[T any](msg string, items map[string]T, summary func(string, T) string) {
	labels := fiware.SummaryOf(items, summary)
	fmt.Printf("%s '%s'\n", msg, strings.Join(labels, "','"))
}

func listMessage[T any](msg string, items []T, summary func(T) string) {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, summary(item))
	}
	fmt.Printf("%s '%s'\n", msg, strings.Join(labels, "','"))
}
