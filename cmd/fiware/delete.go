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
)

var canDelete []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
	"entities",
}

func deleteResource(c *cli.Context, store *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canPost, ", "))
	}

	selected, err := getConfig(c, store)
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
		var header http.Header
		switch arg {
		case "devices":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := deleteDevices(selected, client, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := deleteServices(selected, client, header, vertical); err != nil {
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
			useDescription := c.Bool(useDescriptionFlag.Name)
			if err := deleteSuscriptions(selected, client, header, vertical, useDescription); err != nil {
				return err
			}
		case "rules":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := deleteRules(selected, client, header, vertical); err != nil {
				return err
			}
		case "entities":
			if _, header, err = getKeystoneHeaders(c, selected); err != nil {
				return err
			}
			if err := deleteEntities(selected, client, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to delete resource %s", arg)
		}
	}
	return nil
}

func deleteDevices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("DELETing devices with IDs", vertical.Devices,
		func(g fiware.Device) string { return g.DeviceId },
	)
	return api.DeleteDevices(client, header, vertical.Devices)
}

func deleteServices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("DELETing groups with API Keys", vertical.Services,
		func(g fiware.Service) string { return g.APIKey },
	)
	return api.DeleteServices(client, header, vertical.Services)
}

func deleteSuscriptions(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical, useDescription bool) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	if !useDescription {
		listMessage("DELETing suscriptions with IDs", vertical.Suscriptions,
			func(g fiware.Suscription) string { return g.ID },
		)
	} else {
		listMessage("DELETing suscriptions with ids (or descriptions)", vertical.Suscriptions,
			func(g fiware.Suscription) string { return fmt.Sprintf("%s (%s)", g.ID, g.Description) },
		)
	}
	return api.DeleteSuscriptions(client, header, vertical.Suscriptions, useDescription)
}

func deleteRules(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	listMessage("DELETing rules with names", vertical.Rules,
		func(g fiware.Rule) string { return g.Name },
	)
	return api.DeleteRules(client, header, vertical.Rules)
}

func knownEntities(vertical fiware.Vertical) []fiware.Entity {
	knownTypes := make(map[string]struct{})
	for _, entType := range vertical.EntityTypes {
		knownTypes[entType.Type] = struct{}{}
	}
	knownEntities := make([]fiware.Entity, 0, len(vertical.Entities))
	for _, current := range vertical.Entities {
		if _, match := knownTypes[current.Type]; match {
			knownEntities = append(knownEntities, current)
		}
	}
	return knownEntities
}

func deleteEntities(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical fiware.Vertical) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	toDelete := knownEntities(vertical)
	listMessage("DELETing entities ", toDelete,
		func(g fiware.Entity) string { return strings.Join([]string{g.Type, g.ID}, "/") },
	)
	return api.DeleteEntities(client, header, toDelete)
}
