package snapshots

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
)

type Urbo struct {
	Selected  config.Config
	Urbo      *urbo.Urbo
	Headers   http.Header
	Verticals map[string]fiware.Vertical
}

func NewUrbo(selected config.Config, client keystone.HTTPClient, api *urbo.Urbo, headers http.Header) (*Urbo, error) {
	downloader := &Urbo{
		Selected:  selected,
		Urbo:      api,
		Headers:   headers,
		Verticals: make(map[string]fiware.Vertical),
	}
	verticals, err := api.GetVerticals(client, headers)
	if err != nil {
		return nil, err
	}
	downloader.Verticals = verticals
	return downloader, nil
}

// List all available verticals as strings "name (slug)"
func (v *Urbo) List() ([]string, error) {
	names := make([]string, 0, len(v.Verticals))
	for slug, vertical := range v.Verticals {
		names = append(names, fmt.Sprintf("%s (%s)", vertical.Name, slug))
	}
	sort.Sort(sort.StringSlice(names))
	return names, nil
}

// Dowload all panels in vertical, return vertical manifest and panels indexed by slug
func (d *Urbo) Snap(client keystone.HTTPClient, v fiware.Vertical) (fiware.Manifest, map[string]json.RawMessage, error) {
	result := fiware.Manifest{
		Subservice: d.Selected.Subservice,
	}
	clean_vertical, err := d.Urbo.GetVertical(client, d.Headers, v.Slug)
	if err != nil {
		return result, nil, err
	}
	clean_vertical.UrboVerticalStatus = fiware.UrboVerticalStatus{}
	result.Verticals = map[string]fiware.Vertical{
		v.Slug: clean_vertical,
	}
	panels := make(map[string]json.RawMessage)
	for _, panel := range clean_vertical.AllPanels() {
		content, err := d.Urbo.DownloadPanel(client, d.Headers, panel)
		if err != nil {
			return result, nil, err
		}
		panels[panel] = content
	}
	return result, panels, nil
}
