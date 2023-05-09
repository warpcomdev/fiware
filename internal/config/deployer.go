package config

// Urbo-deployer Environment model
type Environment struct {
	EnvironmentName string            `json:"environmentName"`
	EnvironmentType string            `json:"environmentType"`
	Customer        string            `json:"customer"`
	Service         string            `json:"service"`
	Database        string            `json:"database"`
	JenkinsLabel    string            `json:"jenkinsLabel"`
	JenkinsFolder   string            `json:"jenkinsFolder"`
	BIConnection    string            `json:"biConnection"`
	DatabaseSchemas map[string]string `json:"databaseSchemas"`
	Api             struct {
		Postgis      string `json:"postgis"`
		Orchestrator string `json:"orchestrator"`
		Orion        string `json:"orion"`
		Keystone     string `json:"keystone"`
		Perseo       string `json:"perseo"`
		Pentaho      string `json:"pentaho"`
		Urbo         string `json:"urbo"`
		Jenkins      string `json:"jenkins"`
	} `json:"api"`
	NotificationEndpoints map[string]string `json:"notificationEndpoints"`
}

func FromConfig(cfg Config) Environment {
	result := Environment{
		EnvironmentName:       cfg.Name,
		EnvironmentType:       cfg.Type,
		Customer:              cfg.Customer,
		Service:               cfg.Service,
		Database:              cfg.Database,
		JenkinsLabel:          cfg.JenkinsLabel,
		JenkinsFolder:         cfg.JenkinsFolder,
		BIConnection:          cfg.BIConnection,
		DatabaseSchemas:       make(map[string]string),
		NotificationEndpoints: make(map[string]string),
	}
	result.DatabaseSchemas["DEFAULT"] = cfg.Schema
	result.Api.Jenkins = cfg.JenkinsURL
	result.Api.Postgis = cfg.PostgisURL
	result.Api.Orchestrator = cfg.OrchURL
	result.Api.Orion = cfg.OrionURL
	result.Api.Keystone = cfg.KeystoneURL
	result.Api.Perseo = cfg.PerseoURL
	result.Api.Pentaho = cfg.PentahoURL
	result.Api.Urbo = cfg.UrboURL
	for key, val := range EndpointsFromParams(cfg.Params) {
		result.NotificationEndpoints[key] = val
	}
	return result
}

func EndpointsFromParams(cfgParams map[string]string) map[string]string {
	result := make(map[string]string)
	for param, endpoint := range map[string]string{
		"cygnus_url":          "HISTORIC",
		"cygnus_url_lastdata": "LASTDATA",
		"lastdata_url":        "LASTDATA",
		"perseo_url":          "RULES",
		"HISTORIC":            "HISTORIC",
		"LASTDATA":            "LASTDATA",
		"RULES":               "RULES",
	} {
		if value, ok := cfgParams[param]; ok {
			result[endpoint] = value
		}
	}
	return result
}
