package snapshots

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
	"github.com/warpcomdev/fiware/models"
)

// List all available verticals as strings "name (slug)"
func VerticalList(verticals map[string]models.Vertical) []string {
	names := make([]string, 0, len(verticals))
	for slug, vertical := range verticals {
		names = append(names, fmt.Sprintf("%s (%s)", vertical.Name, slug))
	}
	sort.Strings(names)
	return names
}

// Dowload all panels in vertical, return vertical manifest and panels indexed by slug
func Urbo(client keystone.HTTPClient, api *urbo.Urbo, selected config.Config, headers http.Header, v models.Vertical) (models.Manifest, map[string]json.RawMessage, error) {
	result := models.Manifest{
		Subservice: selected.Subservice,
	}
	clean_vertical, err := api.GetVertical(client, headers, v.Slug)
	if err != nil {
		return result, nil, err
	}
	clean_vertical.UrboVerticalStatus = models.UrboVerticalStatus{}
	result.Verticals = map[string]models.Vertical{
		v.Slug: clean_vertical,
	}
	panels := make(map[string]json.RawMessage)
	for _, panel := range clean_vertical.AllPanels() {
		content, err := api.DownloadPanel(client, headers, panel)
		if err != nil {
			return result, nil, err
		}
		panels[panel] = content
	}
	return result, panels, nil
}
