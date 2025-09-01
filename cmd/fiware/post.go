package main

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/perseo"
	"github.com/warpcomdev/fiware/internal/urbo"
	"github.com/warpcomdev/fiware/iotam"
	"github.com/warpcomdev/fiware/keystone"
	"github.com/warpcomdev/fiware/models"
	"github.com/warpcomdev/fiware/orion"
)

var canPost []string = []string{
	"services",
	"devices",
	"suscriptions",
	"rules",
	"entities",
	"verticals",
	"users",
	"usergroups",
	"projects",
}

func filterEntities(c *cli.Context, manifest models.Manifest) (models.Manifest, error) {
	filterType := c.String(filterTypeFlag.Name)
	postedEntities := make([]models.Entity, 0, len(manifest.Entities))
	for _, entity := range manifest.Entities {
		if filterType == "" || filterType == entity.Type {
			postedEntities = append(postedEntities, entity)
		}
	}
	if len(postedEntities) <= 0 {
		return models.Manifest{}, fmt.Errorf("no entities of type %s found", filterType)
	}
	postedManifest := manifest
	postedManifest.Entities = postedEntities
	return postedManifest, nil
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

	batchSize := c.Int(batchSizeFlag.Name)
	overrideMetadata := c.Bool(overrideMetadataFlag.Name)
	useDescription := !c.Bool(useExactIdFlag.Name)
	client := httpClient(verbosity(c), configuredTimeout(c))
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
			filterManifest, err := filterEntities(c, manifest)
			if err != nil {
				return err
			}
			if err := postEntities(selected, client, header, filterManifest, batchSize, overrideMetadata); err != nil {
				return err
			}
		case "users":
			var k *keystone.Keystone
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postUsers(k, client, header, manifest); err != nil {
				return err
			}
		case "usergroups":
			var k *keystone.Keystone
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postGroups(k, client, header, manifest); err != nil {
				return err
			}
		case "projects":
			var k *keystone.Keystone
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := postProjects(k, client, header, manifest); err != nil {
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

func postDevices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical models.Manifest) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing devices with IDs", vertical.Devices,
		func(g models.Device) string { return g.DeviceId },
	)
	return api.PostDevices(client, header, vertical.Devices)
}

func postServices(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical models.Manifest) error {
	api, err := iotam.New(ctx.IotamURL)
	if err != nil {
		return err
	}
	listMessage("POSTing groups with API Keys", vertical.Services,
		func(g models.Service) string { return g.APIKey },
	)
	return api.PostServices(client, header, vertical.Services)
}

func postSuscriptions(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical models.Manifest, useDescription bool) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	dictMessage("POSTing suscriptions with descriptions", vertical.Subscriptions,
		func(k string, v models.Subscription) string { return v.Description },
	)
	// Mewrge configuration notificationEndpoints with vertical ones
	ep := config.FromConfig(ctx).NotificationEndpoints
	for k, v := range vertical.Environment.NotificationEndpoints {
		ep[k] = v
	}
	subs := slices.Collect(maps.Values(vertical.Subscriptions))
	return api.PostSuscriptions(client, header, subs, ep, useDescription)
}

func postRules(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical models.Manifest) error {
	api, err := perseo.New(ctx.PerseoURL)
	if err != nil {
		return err
	}
	dictMessage("POSTing rules with names", vertical.Rules,
		func(k string, v models.Rule) string {
			if v.Name != "" {
				return v.Name
			}
			return k
		},
	)
	return api.PostRules(client, header, slices.Collect(maps.Values(vertical.Rules)))
}

func postUsers(k *keystone.Keystone, client keystone.HTTPClient, header http.Header, vertical models.Manifest) error {
	listMessage("POSTing users with names ", vertical.Users,
		func(u models.User) string { return u.Name })
	return k.PostUsers(client, header, vertical.Users)
}

func postGroups(k *keystone.Keystone, client keystone.HTTPClient, header http.Header, vertical models.Manifest) error {
	listMessage("POSTing groups with names ", vertical.Groups,
		func(g models.Group) string { return g.Name })
	return k.PostGroups(client, header, vertical.Groups)
}

func postProjects(k *keystone.Keystone, client keystone.HTTPClient, header http.Header, vertical models.Manifest) error {
	listMessage("POSTing projects with names ", vertical.Projects,
		func(p models.Project) string { return p.Name })
	return k.PostProjects(client, header, vertical.Projects)
}

func postEntities(ctx config.Config, client keystone.HTTPClient, header http.Header, vertical models.Manifest, batchSize int, overrideMetadata bool) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	merged := orion.Merge(vertical.EntityTypes, vertical.Entities)
	listMessage("POSTing entities with names", merged,
		func(g orion.Entity) string { return fmt.Sprintf("%s/%s", g.Type(), g.ID()) },
	)
	return api.UpdateEntities(client, header, merged, batchSize, overrideMetadata)
}

func postVerticals(ctx config.Config, client keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical models.Manifest) error {
	dictMessage("POSTing verticals with slugs", vertical.Verticals,
		func(k string, v models.Vertical) string { return v.Slug },
	)
	return u.PostVerticals(client, header, vertical.Verticals)
}

func dictMessage[T any](msg string, items map[string]T, summary func(string, T) string) {
	labels := models.SummaryOf(items, summary)
	fmt.Printf("%s '%s'\n", msg, strings.Join(labels, "','"))
}

func listMessage[T any](msg string, items []T, summary func(T) string) {
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, summary(item))
	}
	fmt.Printf("%s '%s'\n", msg, strings.Join(labels, "','"))
}
