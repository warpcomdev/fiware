package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/keystone"
)

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
	userAssignments := make([]fiware.RoleAssignment, 0, len(vertical.Assignments))
	for _, assign := range vertical.Assignments {
		if assign.User.ID != "" {
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
	return errors.New("TODO: Not implemented")
}
