package snapshots

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/template"
)

// Write all assets in manifest using deployer format.
// panels might containe payload for any panel in manifest.Verticals.
// panels outside manifest.Verticals are not stored.
func WriteManifest(manifest fiware.Manifest, panels map[string]json.RawMessage, folder string) (fiware.ManifestSource, error) {
	result := fiware.ManifestSource{
		Files: make([]string, 0, 8),
	}

	// dump assets
	conditionalSave := func(asset string, when bool, manifest fiware.Manifest) error {
		if when {
			filename := asset + ".json"
			if err := config.AtomicSave(filepath.Join(folder, filename), asset, manifest); err != nil {
				return err
			}
			result.Files = append(result.Files, filename)
		}
		return nil
	}

	if err := conditionalSave("rules", len(manifest.Rules) > 0, fiware.Manifest{
		Rules: manifest.Rules,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("subs", len(manifest.Subscriptions) > 0, fiware.Manifest{
		Subscriptions: manifest.Subscriptions,
		Environment:   manifest.Environment,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("registrations", len(manifest.Registrations) > 0, fiware.Manifest{
		Registrations: manifest.Registrations,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("groups", len(manifest.Services) > 0, fiware.Manifest{
		Services: manifest.Services,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("devices", len(manifest.Devices) > 0, fiware.Manifest{
		Devices: manifest.Devices,
	}); err != nil {
		return result, err
	}

	if len(panels) > 0 {
		manifest := fiware.Manifest{
			Verticals: manifest.Verticals,
			ManifestPanels: fiware.PanelManifest{
				Sources: make(map[string]fiware.ManifestSource),
			},
		}
		// Only dump panels in verticals
		for verticalSlug, vertical := range manifest.Verticals {
			panelSource := fiware.ManifestSource{
				Path:  "./panels",
				Files: make([]string, 0, len(panels)),
			}
			for _, slug := range vertical.AllPanels() {
				payload, ok := panels[slug]
				if !ok {
					continue
				}
				filename := slug + ".json"
				panelSource.Files = append(panelSource.Files, filename)
				fullPath := filepath.Join(folder, "panels", filename)
				if err := config.AtomicSaveBytes(fullPath, slug, payload); err != nil {
					return result, err
				}
			}
			manifest.ManifestPanels.Sources[verticalSlug] = panelSource
		}
		if err := conditionalSave("verticals", true, manifest); err != nil {
			return result, err
		}
	} else {
		if err := conditionalSave("verticals", len(manifest.Verticals) > 0, fiware.Manifest{
			Verticals: manifest.Verticals,
		}); err != nil {
			return result, err
		}
	}

	// dump entities as CSV
	if len(manifest.EntityTypes) > 0 {
		entityManifest := fiware.Manifest{
			EntityTypes: manifest.EntityTypes,
			Entities:    manifest.Entities,
		}
		csvData := &bytes.Buffer{}
		plain, err := template.ManifestForTemplate(entityManifest, nil)
		if err != nil {
			return result, err
		}
		if err := template.Render([]string{"default_csv.tmpl"}, plain, csvData); err != nil {
			return result, err
		}
		if err := config.AtomicSaveBytes(filepath.Join(folder, "entities.csv"), "entities", csvData.Bytes()); err != nil {
			return result, err
		}
	}
	return result, nil
}
