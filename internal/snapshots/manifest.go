package snapshots

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/template"
	"github.com/warpcomdev/fiware/models"
)

// Write all assets in manifest using deployer format.
// panels might containe payload for any panel in manifest.Verticals.
// panels outside manifest.Verticals are not stored.
func WriteManifest(manifest models.Manifest, panels map[string]json.RawMessage, writer config.Writer) (models.ManifestSource, error) {
	result := models.ManifestSource{
		Files: make([]string, 0, 8),
	}

	// dump assets
	conditionalSave := func(asset string, when bool, manifest models.Manifest) error {
		if when {
			filename := asset + ".json"
			if err := config.AtomicSave(writer, filename, asset, manifest); err != nil {
				return err
			}
			result.Files = append(result.Files, filename)
		}
		return nil
	}

	if err := conditionalSave("rules", len(manifest.Rules) > 0, models.Manifest{
		Rules: manifest.Rules,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("subs", len(manifest.Subscriptions) > 0, models.Manifest{
		Subscriptions: manifest.Subscriptions,
		Environment:   manifest.Environment,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("registrations", len(manifest.Registrations) > 0, models.Manifest{
		Registrations: manifest.Registrations,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("groups", len(manifest.Services) > 0, models.Manifest{
		Services: manifest.Services,
	}); err != nil {
		return result, err
	}

	if err := conditionalSave("devices", len(manifest.Devices) > 0, models.Manifest{
		Devices: manifest.Devices,
	}); err != nil {
		return result, err
	}

	if len(panels) > 0 {
		manifest := models.Manifest{
			Verticals: manifest.Verticals,
			ManifestPanels: models.PanelManifest{
				Sources: make(map[string]models.ManifestSource),
			},
		}
		// Only dump panels in verticals
		for verticalSlug, vertical := range manifest.Verticals {
			panelSource := models.ManifestSource{
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
				fullPath := filepath.Join("panels", filename)
				if err := writer.AtomicSave(fullPath, slug, payload); err != nil {
					return result, err
				}
			}
			manifest.ManifestPanels.Sources[verticalSlug] = panelSource
		}
		if err := conditionalSave("verticals", true, manifest); err != nil {
			return result, err
		}
	} else {
		if err := conditionalSave("verticals", len(manifest.Verticals) > 0, models.Manifest{
			Verticals: manifest.Verticals,
		}); err != nil {
			return result, err
		}
	}

	// dump entities as CSV
	if len(manifest.EntityTypes) > 0 {
		entityManifest := models.Manifest{
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
		if err := writer.AtomicSave("entities.csv", "entities", csvData.Bytes()); err != nil {
			return result, err
		}
	}
	return result, nil
}
