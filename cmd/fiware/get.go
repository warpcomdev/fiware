package main

import (
	"errors"
	"fmt"
	"log"
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
	"domains",
	"panels",
	"verticals",
	"entities",
	"registrations",
	"users",
	"usergroups",
	"roles",
	"userroles",
	"grouproles",
	"rolemap",
}

type serializerWithSetup interface {
	serialize.Serializer
	Setup(serialize.Writer, map[string]string)
	Begin()
	End()
}

func getConfig(c *cli.Context, store *config.Store) (zero config.Config, err error) {
	if err := store.Read(""); err != nil {
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

// Might update selected if subservice flag is set
func getKeystoneHeaders(c *cli.Context, selected *config.Config) (k *keystone.Keystone, h http.Header, err error) {
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

// Might update "Selected" overwriting subservice, if set in flag
func getUrboHeaders(c *cli.Context, selected *config.Config) (u *urbo.Urbo, h http.Header, err error) {
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
	header := u.Headers(token)
	return u, header, nil
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

	vertical := &fiware.Manifest{
		Subservice: selected.Subservice,
		Environment: fiware.Environment{
			NotificationEndpoints: config.FromConfig(selected).NotificationEndpoints,
		},
	}
	maximum := c.Int(maxFlag.Name)
	client := httpClient(verbosity(c), configuredTimeout(c))
	for _, arg := range c.Args().Slice() {
		var k *keystone.Keystone
		var u *urbo.Urbo
		var header http.Header
		switch arg {
		case "devices":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getDevices(selected, client, header, vertical); err != nil {
				return err
			}
		case "services":
			fallthrough
		case "groups":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
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
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getSuscriptions(selected, client, header, vertical); err != nil {
				return err
			}
		case "registrations":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getRegistrations(selected, client, header, vertical); err != nil {
				return err
			}
		case "entities":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			filterId := c.String(filterIdFlag.Name)
			filterType := c.String(filterTypeFlag.Name)
			simpleQuery := c.String(simpleQueryFlag.Name)
			if err := getEntities(selected, client, header, filterId, filterType, simpleQuery, maximum, vertical); err != nil {
				return err
			}
		case "rules":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getRules(selected, client, header, vertical); err != nil {
				return err
			}
		case "projects":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getProjects(selected, client, k, header, vertical); err != nil {
				return err
			} else {
				// Take the chance to store the project list in cache,
				// so we can use it for autocomplete later on.
				// Notice we only do this when called through the CLI.
				// getResource is also called from the API.
				refreshCache := func() error {
					if err := cacheProjects(&selected, vertical.Projects); err != nil {
						return err
					}
					// Our selected store might have local changes due to
					// flags, so it's better to load a fresh copy of the config
					// and make sure we just update the cache.
					orig, err := store.Info(selected.Name)
					if err != nil {
						return err
					}
					orig.ProjectCache = selected.ProjectCache
					if err := store.Save(orig); err != nil {
						return err
					}
					return nil
				}
				// This is best effort, so we don't care if it fails.
				if err := refreshCache(); err != nil {
					log.Printf("failed to refresh cache of %s: %s", selected.Name, err)
				}
			}
		case "domains":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getDomains(selected, client, k, header, vertical); err != nil {
				return err
			}
		case "users":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getUsers(selected, client, k, header, vertical); err != nil {
				return err
			}
		case "ug":
			fallthrough
		case "usergroups":
			fallthrough
		case "user_groups":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getGroups(selected, client, k, header, vertical); err != nil {
				return err
			}
		case "roles":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getRoles(selected, client, k, header, vertical); err != nil {
				return err
			}
		case "ur":
			fallthrough
		case "userroles":
			fallthrough
		case "user_roles":
			userIds := c.StringSlice(userIdFlag.Name)
			if len(userIds) <= 0 {
				return errors.New("no user id provided")
			}
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getUserRoles(selected, client, k, header, userIds, vertical); err != nil {
				return err
			}
		case "gr":
			fallthrough
		case "grouproles":
			fallthrough
		case "group_roles":
			groupIds := c.StringSlice(groupIdFlag.Name)
			if len(groupIds) <= 0 {
				return errors.New("no group id provided")
			}
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getGroupRoles(selected, client, k, header, groupIds, vertical); err != nil {
				return err
			}
		case "rolemap":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := getRolemap(selected, client, k, header, vertical); err != nil {
				return err
			}
		case "panels":
			if u, header, err = getUrboHeaders(c, &selected); err != nil {
				return err
			}
			if err := getPanels(selected, client, u, header, vertical); err != nil {
				return err
			}
		case "verticals":
			if u, header, err = getUrboHeaders(c, &selected); err != nil {
				return err
			}
			if err := getVerticals(selected, client, u, header, vertical); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to get resource %s", arg)
		}
	}

	// This is mostly cosmetic, but it might induce to error if an user
	// specifies --subservice X and doesn't see the subservice in the output.
	if subservice := c.String(subServiceFlag.Name); subservice != "" {
		vertical.Subservice = subservice
	}
	return output.Encode(outfile, vertical, selected.Params)
}

func getDevices(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Manifest) error {
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

func getServices(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Manifest) error {
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

func getSuscriptions(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Manifest) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	if vertical.Environment.NotificationEndpoints == nil {
		vertical.Environment.NotificationEndpoints = make(map[string]string)
	}
	suscriptions, err := api.Subscriptions(c, header, vertical.Environment.NotificationEndpoints)
	if err != nil {
		return err
	}
	vertical.Subscriptions = orion.SubsMap(suscriptions)
	return nil
}

func getRegistrations(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Manifest) error {
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

func getEntities(ctx config.Config, c keystone.HTTPClient, header http.Header, filterId, filterType, simpleQuery string, maximum int, vertical *fiware.Manifest) error {
	api, err := orion.New(ctx.OrionURL)
	if err != nil {
		return err
	}
	types, values, err := api.Entities(c, header, filterId, filterType, simpleQuery, maximum)
	if err != nil {
		return err
	}
	vertical.EntityTypes = types
	vertical.Entities = values
	return nil
}

func getRules(ctx config.Config, c keystone.HTTPClient, header http.Header, vertical *fiware.Manifest) error {
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

func getProjects(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Manifest) error {
	projects, err := k.Projects(c, header)
	if err != nil {
		return err
	}
	vertical.Projects = projects
	return nil
}

func getUsers(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Manifest) error {
	users, err := k.Users(c, header)
	if err != nil {
		return err
	}
	vertical.Users = users
	return nil
}

func getGroups(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Manifest) error {
	groups, err := k.Groups(c, header)
	if err != nil {
		return err
	}
	vertical.Groups = groups
	return nil
}

func getRoles(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Manifest) error {
	roles, err := k.Roles(c, header)
	if err != nil {
		return err
	}
	vertical.Roles = roles
	return nil
}

func getUserRoles(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, uids []string, vertical *fiware.Manifest) error {
	assignments, err := k.UserRoles(c, header, uids)
	if err != nil {
		return err
	}
	vertical.Assignments = assignments
	return nil
}

func getGroupRoles(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, gids []string, vertical *fiware.Manifest) error {
	assignments, err := k.GroupRoles(c, header, gids)
	if err != nil {
		return err
	}
	vertical.Assignments = assignments
	return nil
}

func getRolemap(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Manifest) error {
	roleMap, err := k.RoleMap(c, header)
	if err != nil {
		return err
	}
	vertical.Projects = roleMap.Projects
	vertical.Roles = roleMap.Roles
	vertical.Users = roleMap.Users
	vertical.Groups = roleMap.Groups
	return nil
}

func getDomains(ctx config.Config, c keystone.HTTPClient, k *keystone.Keystone, header http.Header, vertical *fiware.Manifest) error {
	domains, err := k.Domains(c, header, false)
	if err != nil {
		return err
	}
	vertical.Domains = domains
	return nil
}

func getPanels(ctx config.Config, c keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical *fiware.Manifest) error {
	panels, err := u.Panels(c, header)
	if err != nil {
		return err
	}
	vertical.Panels = panels
	return nil
}

func getVerticals(ctx config.Config, c keystone.HTTPClient, u *urbo.Urbo, header http.Header, vertical *fiware.Manifest) error {
	verticals, err := u.GetVerticals(c, header)
	if err != nil {
		return err
	}
	vertical.Verticals = verticals
	return nil
}
