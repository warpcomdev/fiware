package importer

import (
	"github.com/warpcomdev/fiware"
)

type deployerConfig struct {
	Environment    fiware.Environment                        `json:"environment"`
	Deployment     fiware.DeploymentManifest                 `json:"deployment"`
	Subscriptions  map[string]map[string]fiware.Subscription `json:"subscriptions"`
	Rules          map[string]map[string]fiware.Rule         `json:"rules"`
	ManifestPanels map[string]fiware.PanelManifest           `json:"panels,omitempty"`
	Verticals      map[string]map[string]fiware.Vertical     `json:"verticals,omitempty"`
}

// Read a deployer config file
func (rawConfig deployerConfig) ToManifest() fiware.Manifest {
	manifest := fiware.Manifest{
		Subscriptions:  make(map[string]fiware.Subscription),
		Rules:          make(map[string]fiware.Rule),
		Verticals:      make(map[string]fiware.Vertical),
		ManifestPanels: fiware.PanelManifest{Sources: make(map[string]fiware.ManifestSource)},
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
