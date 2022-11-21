package config

// Urbo-deployer environment model
type environment struct {
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

func fromConfig(cfg *Config) environment {
	result := environment{
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
	if cygnus, ok := cfg.Params["cygnus_url"]; ok {
		result.NotificationEndpoints["HISTORIC"] = cygnus
	}
	if lastdata, ok := cfg.Params["cygnus_url_lastdata"]; ok {
		result.NotificationEndpoints["LASTDATA"] = lastdata
	}
	if perseo, ok := cfg.Params["perseo_url"]; ok {
		result.NotificationEndpoints["RULES"] = perseo
	}
	return result
}
