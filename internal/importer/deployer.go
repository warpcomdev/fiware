package importer

import (
	"github.com/warpcomdev/fiware/models"
)

type deployerConfig struct {
	Environment    models.Environment                        `json:"environment"`
	Deployment     models.DeploymentManifest                 `json:"deployment"`
	Subscriptions  map[string]map[string]models.Subscription `json:"subscriptions"`
	Rules          map[string]map[string]models.Rule         `json:"rules"`
	ManifestPanels map[string]models.PanelManifest           `json:"panels,omitempty"`
	Verticals      map[string]map[string]models.Vertical     `json:"verticals,omitempty"`
}

// Read a deployer config file
func (rawConfig deployerConfig) ToManifest() models.Manifest {
	manifest := models.Manifest{
		Subscriptions:  make(map[string]models.Subscription),
		Rules:          make(map[string]models.Rule),
		Verticals:      make(map[string]models.Vertical),
		ManifestPanels: models.PanelManifest{Sources: make(map[string]models.ManifestSource)},
	}
	manifest.Environment = rawConfig.Environment
	manifest.Deployment = rawConfig.Deployment
	for section, subs := range rawConfig.Subscriptions {
		for label, sub := range subs {
			manifest.Subscriptions[section+"."+label] = sub
		}
	}
	for section, rules := range rawConfig.Rules {
		for label, rule := range rules {
			manifest.Rules[section+"."+label] = rule
		}
	}
	for section, rules := range rawConfig.Verticals {
		for label, vertical := range rules {
			manifest.Verticals[section+"."+label] = vertical
		}
	}
	for section, panels := range rawConfig.ManifestPanels {
		for label, source := range panels.Sources {
			manifest.ManifestPanels.Sources[section+"."+label] = source
		}
	}
	return manifest
}
