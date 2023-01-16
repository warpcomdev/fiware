package snapshots

import (
	"net/http"
	"sort"
	"strings"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/iotam"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/orion"
	"github.com/warpcomdev/fiware/internal/perseo"
)

type Project struct {
	Selected config.Config
	Keystone *keystone.Keystone
	Headers  http.Header
	Projects []fiware.Project
}

func NewProject(selected config.Config, client keystone.HTTPClient, keystone *keystone.Keystone, headers http.Header) (*Project, error) {
	downloader := &Project{
		Selected: selected,
		Keystone: keystone,
		Headers:  headers,
	}
	projects, err := keystone.Projects(client, headers)
	if err != nil {
		return nil, err
	}
	// Skip all projects that don't begin with "/"
	cursor := 0
	for _, item := range projects {
		if strings.HasPrefix(item.Name, "/") {
			projects[cursor] = item
			cursor += 1
		}
	}
	downloader.Projects = projects[0:cursor]
	return downloader, nil
}

func (p *Project) Names() []string {
	names := make([]string, 0, len(p.Projects))
	for _, project := range p.Projects {
		names = append(names, project.Name)
	}
	sort.Sort(sort.StringSlice(names))
	return names
}

// Snap takes an snapshot of all assets in project
func (p *Project) Snap(client keystone.HTTPClient, project fiware.Project, maximum int) (fiware.Manifest, error) {
	var result fiware.Manifest
	// IMPORTANT! must set selected subservice!
	selected := p.Selected
	headers := p.Headers.Clone()
	selected.Subservice = project.Name
	headers.Set("Fiware-Servicepath", project.Name)

	// Pre-create all clients, to fail early
	orionServer, err := orion.New(selected.OrionURL)
	if err != nil {
		return result, err
	}
	perseoServer, err := perseo.New(selected.PerseoURL)
	if err != nil {
		return result, err
	}
	iotamServer, err := iotam.New(selected.IotamURL)
	if err != nil {
		return result, err
	}

	// Dump orion: entities, subscriptions and registrations
	types, entities, err := orionServer.Entities(client, headers, "", "", "", maximum)
	if err != nil {
		return result, err
	}
	result.EntityTypes = types
	result.Entities = entities
	subs, err := orionServer.Subscriptions(client, headers, config.FromConfig(selected).NotificationEndpoints)
	if err != nil {
		return result, err
	}
	result.Subscriptions = orion.SubsMap(subs)
	regs, err := orionServer.Registrations(client, headers)
	if err != nil {
		return result, err
	}
	result.Registrations = regs

	// Dump perseo: rules
	rules, err := perseoServer.Rules(client, headers)
	if err != nil {
		return result, err
	}
	namedRules := make(map[string]fiware.Rule, len(rules))
	for _, rule := range rules {
		namedRules[rule.Name] = rule
	}
	result.Rules = namedRules

	// Dump iotam: groups and devices
	groups, err := iotamServer.Services(client, headers)
	if err != nil {
		return result, err
	}
	result.Services = groups
	devices, err := iotamServer.Devices(client, headers)
	if err != nil {
		return result, err
	}
	result.Devices = devices

	return result, nil
}
