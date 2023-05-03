package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type stringError string

func (s stringError) Error() string {
	return string(s)
}

const (
	ErrNoContext        stringError = "no context in use"
	ErrParametersNumber stringError = "please provide parameter - value pairs"
	HiddenToken                     = "***"
)

// Config almacena información de conexión a un entorno
type Config struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Customer      string            `json:"customer"`
	KeystoneURL   string            `json:"keystone"`
	OrionURL      string            `json:"orion"`
	IotamURL      string            `json:"iotam"`
	PerseoURL     string            `json:"perseo"`
	UrboURL       string            `json:"urbo"`
	OrchURL       string            `json:"orch"`
	PostgisURL    string            `json:"postgis"`
	JenkinsURL    string            `json:"jenkins"`
	PentahoURL    string            `json:"pentaho"`
	Service       string            `json:"service"`
	Subservice    string            `json:"subservice"`
	Database      string            `json:"database"`
	Schema        string            `json:"schema"`
	Username      string            `json:"username"`
	JenkinsLabel  string            `json:"jenkinsLabel"`
	JenkinsFolder string            `json:"jenkinsFolder"`
	BIConnection  string            `json:"biConnection"`
	Token         string            `json:"token,omitempty"`
	UrboToken     string            `json:"urbotoken,omitempty"`
	Params        map[string]string `json:"params,omitempty"`
	ProjectCache  []string          `json:"projects,omitempty"`
}

func (c *Config) defaults() {
	if c.Database == "" {
		c.Database = "urbo2"
	}
	if c.Schema == "" {
		c.Schema = c.Service
	}
	if c.Type == "" {
		c.Type = "DEV"
	}
	if c.Customer == "" {
		c.Customer = c.Name
	}
	if c.JenkinsLabel == "" {
		if c.JenkinsFolder != "" {
			c.JenkinsLabel = c.JenkinsFolder
		} else {
			c.JenkinsLabel = c.Customer
		}
	}
	if c.JenkinsFolder == "" {
		if c.JenkinsLabel != "" {
			c.JenkinsFolder = c.JenkinsLabel
		} else {
			c.JenkinsFolder = c.Customer
		}
	}
	if c.BIConnection == "" {
		c.BIConnection = c.Service
	}
}

func writePair(w *bufio.Writer, sep1, k, sep2, v string) {
	w.WriteString(sep1)
	w.WriteString(k)
	w.WriteString(sep2)
	fmt.Fprintf(w, "%q", v) // in case it contains invalid chars
}

// Pairs of config key and value
func (c *Config) pairs() map[string]*string {
	p := map[string]*string{
		"type":          &c.Type,
		"customer":      &c.Customer,
		"keystone":      &c.KeystoneURL,
		"orion":         &c.OrionURL,
		"iotam":         &c.IotamURL,
		"perseo":        &c.PerseoURL,
		"urbo":          &c.UrboURL,
		"orch":          &c.OrchURL,
		"postgis":       &c.PostgisURL,
		"jenkins":       &c.JenkinsURL,
		"pentaho":       &c.PentahoURL,
		"service":       &c.Service,
		"subservice":    &c.Subservice,
		"database":      &c.Database,
		"schema":        &c.Schema,
		"jenkinsLabel":  &c.JenkinsLabel,
		"jenkinsFolder": &c.JenkinsFolder,
		"biConnection":  &c.BIConnection,
		"username":      &c.Username,
	}
	return p
}

func SortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))
	return keys
}

// Pairs return all context parameters as a map of strings
func (c *Config) Pairs() map[string]string {
	result := make(map[string]string)
	for k, v := range c.pairs() {
		result[k] = *v
	}
	if c.Token != "" {
		result["token"] = HiddenToken
	}
	if c.UrboToken != "" {
		result["urboToken"] = HiddenToken
	}
	result["name"] = c.Name
	return result
}

func (c *Config) String() string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	e := json.NewEncoder(w)
	pairs := c.Pairs()
	w.WriteString("{")
	sep := "\n  \""
	sortedPairs := SortedKeys(pairs)
	for _, label := range sortedPairs {
		value := pairs[label]
		writePair(w, sep, label, "\": ", value)
		sep = ",\n  \""
	}
	if len(c.Params) <= 0 {
		w.WriteString("\n}\n> fiware context set")
	} else {
		w.WriteString(",\n  \"params\": ")
		e.SetIndent("  ", "  ")
		e.Encode(c.Params)
		e.SetIndent("", "")
		// Encode adds a "\n", we don't need to add another
		w.WriteString("}\n> fiware context set")
	}
	for _, label := range sortedPairs {
		value := pairs[label]
		writePair(w, " ", label, " ", value)
	}
	if len(c.Params) > 0 {
		w.WriteString("\n> fiware context params")
		sortedParams := SortedKeys(c.Params)
		for _, k := range sortedParams {
			v := c.Params[k]
			writePair(w, " ", k, " ", v)
		}
	}
	w.WriteString("\n")
	w.Flush()
	return string(buffer.Bytes())
}

func (c *Config) HasToken() string {
	if c.Token == HiddenToken {
		return ""
	}
	return c.Token
}

func (c *Config) HasUrboToken() string {
	if c.UrboToken == HiddenToken {
		return ""
	}
	return c.UrboToken
}

func (c *Config) SetCredentials(token, urboToken string) {
	c.Token = token
	c.UrboToken = urboToken
}
