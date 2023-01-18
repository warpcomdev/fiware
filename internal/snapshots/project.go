package snapshots

import (
	"net/http"
	"sort"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/iotam"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/orion"
	"github.com/warpcomdev/fiware/internal/perseo"
)

func ProjectList(projects []fiware.Project) []string {
	names := make([]string, 0, len(projects))
	for _, project := range projects {
		names = append(names, project.Name)
	}
	sort.Sort(sort.StringSlice(names))
	return names
}

// Snap takes an snapshot of all assets in project
func Project(client keystone.HTTPClient, api *keystone.Keystone, selected config.Config, headers http.Header, project fiware.Project, assets []string, maximum int) (fiware.Manifest, error) {
	var result fiware.Manifest
	assetMap := map[string]bool{
		"entities":      true,
		"subscriptions": true,
		"registrations": true,
		"rules":         true,
		"services":      true,
		"devices":       true,
	}

	// IMPORTANT! must set selected subservice!
	selected.Subservice = project.Name
	headers = headers.Clone()
	headers.Set("Fiware-Servicepath", project.Name)

	// If some assets selected, build assetMap
	if assets != nil {
		for k, _ := range assetMap {
			assetMap[k] = false
		}
		for _, asset := range assets {
			assetMap[asset] = true
		}
	}

	// Pre-create all clients, to fail early
	var (
		orionServer  *orion.Orion
		perseoServer *perseo.Perseo
		iotamServer  *iotam.Iotam
		err          error
	)
	if assetMap["entities"] || assetMap["subscriptions"] || assetMap["registrations"] {
		if orionServer, err = orion.New(selected.OrionURL); err != nil {
			return result, err
		}
	}
	if assetMap["rules"] {
		if perseoServer, err = perseo.New(selected.PerseoURL); err != nil {
			return result, err
		}
	}
	if assetMap["services"] || assetMap["devices"] {
		if iotamServer, err = iotam.New(selected.IotamURL); err != nil {
			return result, err
		}
	}

	// Dump orion: entities, subscriptions and registrations
	if assetMap["entities"] {
		types, entities, err := orionServer.Entities(client, headers, "", "", "", maximum)
		if err != nil {
			return result, err
		}
		result.EntityTypes = types
		result.Entities = entities
	}
	if assetMap["subscriptions"] {
		subs, err := orionServer.Subscriptions(client, headers, config.FromConfig(selected).NotificationEndpoints)
		if err != nil {
			return result, err
		}
		result.Subscriptions = orion.SubsMap(subs)
	}
	if assetMap["registrations"] {
		regs, err := orionServer.Registrations(client, headers)
		if err != nil {
			return result, err
		}
		result.Registrations = regs
	}

	// Dump perseo: rules
	if assetMap["rules"] {
		rules, err := perseoServer.Rules(client, headers)
		if err != nil {
			return result, err
		}
		namedRules := make(map[string]fiware.Rule, len(rules))
		for _, rule := range rules {
			namedRules[rule.Name] = rule
		}
		result.Rules = namedRules
	}

	// Dump iotam: groups and devices
	if assetMap["services"] {
		groups, err := iotamServer.Services(client, headers)
		if err != nil {
			return result, err
		}
		result.Services = groups
	}
	if assetMap["devices"] {
		devices, err := iotamServer.Devices(client, headers)
		if err != nil {
			return result, err
		}
		result.Devices = devices
	}

	return result, nil
}
