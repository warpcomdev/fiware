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
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/keystone"
)

// RoleMap contains the information needed to make a migration
type RoleMap struct {
	Projects []fiware.Project
	Roles    []fiware.Role
	Users    []fiware.User
	Groups   []fiware.Group
	// Populated ID maps
	ProjectToID map[string]string
	RoleToID    map[string]string
	UserToID    map[string]string
	GroupToID   map[string]string
}

func newRoleMap(v fiware.Manifest) (RoleMap, error) {
	m := RoleMap{
		Projects:    v.Projects,
		Roles:       v.Roles,
		Users:       v.Users,
		Groups:      v.Groups,
		ProjectToID: make(map[string]string, len(v.Projects)),
		RoleToID:    make(map[string]string, len(v.Roles)),
		UserToID:    make(map[string]string, len(v.Users)),
		GroupToID:   make(map[string]string, len(v.Groups)),
	}
	for _, project := range m.Projects {
		if project.Name == "" {
			return RoleMap{}, errors.New("project name is empty")
		}
		m.ProjectToID[project.Name] = project.ID
	}
	for _, role := range m.Roles {
		if role.Name == "" {
			return RoleMap{}, errors.New("role name is empty")
		}
		m.RoleToID[role.Name] = role.ID
	}
	for _, user := range m.Users {
		if user.Name == "" {
			return RoleMap{}, errors.New("user name is empty")
		}
		m.UserToID[user.Name] = user.ID
	}
	for _, group := range m.Groups {
		if group.Name == "" {
			return RoleMap{}, errors.New("group name is empty")
		}
		m.GroupToID[group.Name] = group.ID
	}
	return m, nil
}

var canMigrate []string = []string{
	"userroles",
}

func migrateResource(c *cli.Context, config *config.Store) error {
	if c.NArg() <= 0 {
		return fmt.Errorf("select a resource from: %s", strings.Join(canMigrate, ", "))
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

	srcMapPath := c.String(srcMapFlag.Name)
	srcMap, err := importer.Load(srcMapPath, selected.Params, libpath)
	if err != nil {
		return err
	}

	dstMapPath := c.String(dstMapFlag.Name)
	dstMap, err := importer.Load(dstMapPath, selected.Params, libpath)
	if err != nil {
		return err
	}

	client := httpClient(verbosity(c), configuredTimeout(c))
	for _, arg := range c.Args().Slice() {
		var k *keystone.Keystone
		var header http.Header
		switch arg {
		case "userroles":
			if k, header, err = getKeystoneHeaders(c, &selected); err != nil {
				return err
			}
			if err := migrateUserRoles(k, client, header, manifest, srcMap, dstMap); err != nil {
				return err
			}
		default:
			return fmt.Errorf("don't know how to migrate resource %s", arg)
		}
	}
	return nil
}

func migrateUserRoles(k *keystone.Keystone, client keystone.HTTPClient, header http.Header, vertical, srcmap, dstmap fiware.Manifest) error {
	dstRoleMap, err := newRoleMap(dstmap)
	if err != nil {
		return err
	}
	userAssignments := make([]fiware.RoleAssignment, 0, len(vertical.Assignments))
	for _, assign := range vertical.Assignments {
		if assign.User.ID != "" {
			dstId, ok := dstRoleMap.UserToID[assign.User.Name]
			if !ok {
				log.Printf("User %s id not found in destination rolemap, skipping", assign.User.Name)
				continue
			}
			assign.User.ID = dstId
			dstId, ok = dstRoleMap.RoleToID[assign.Role.Name]
			if !ok {
				log.Printf("Role %s id not found in destination rolemap, skipping", assign.Role.Name)
				continue
			}
			assign.Role.ID = dstId
			if assign.ProjectID != "" {
				// Projects do not have inheritance flag
				dstId, ok = dstRoleMap.ProjectToID[assign.ScopeName]
				if !ok {
					log.Printf("Project %s id not found in destination rolemap, skipping", assign.ScopeName)
					continue
				}
				assign.ProjectID = dstId
			}
			userAssignments = append(userAssignments, assign)
		}
	}
	listMessage("Migrating roles ", userAssignments,
		func(a fiware.RoleAssignment) string {
			if a.Inherited != "" {
				return fmt.Sprintf("%s [%s: %s] OS-INHERIT: %s", a.User.Name, a.ScopeName, a.Role.Name, a.Inherited)
			} else {
				return fmt.Sprintf("%s [%s: %s]", a.User.Name, a.ScopeName, a.Role.Name)
			}
		})
	if err := k.PostAssignments(client, header, userAssignments); err != nil {
		return err
	}
	return nil
}
