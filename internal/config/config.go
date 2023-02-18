package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

// Store can manage several configs
type Store struct {
	Path    string // It no longer contains full contexts, only context selector.
	DirPath string // his holds the actual contexts now
	Current Config
}

const (
	tmpContextPrefix = "fiware-context"
	tmpSelectPrefix  = "fiware-selection"
)

// get the proper paths in the new config model
func (s *Store) getConfigDir() (string, error) {
	if s.DirPath != "" {
		return s.DirPath, nil
	}
	s.DirPath = strings.TrimSuffix(s.Path, filepath.Ext(s.Path)) + ".d"
	return s.DirPath, nil
}

func (s *Store) atomicSave(fullPath, tmpPrefix string, data interface{}) error {
	return AtomicSave(FolderWriter(""), fullPath, tmpPrefix, data)
}

// Atomically save a config. Does not change current selection.
func (s *Store) Save(cfg Config) error {
	dirPath, err := s.getConfigDir()
	if err != nil {
		return err
	}
	fullPath := filepath.Join(dirPath, cfg.Name+".json")
	return s.atomicSave(fullPath, tmpContextPrefix, cfg)
}

// Migrate the selection file
func (s *Store) migrate(dirPath string) (string, error) {
	file, err := os.Open(s.Path)
	if err != nil {
		// If the file does not exist, return empty selection
		if os.IsNotExist(err) {
			// unless there is something in the config folder
			options, err := s.listConfigFolder()
			if err != nil {
				return "", err
			}
			// By convention, if there is no selection file,
			// the first file in folder will be selected.
			var selected string
			if len(options) > 0 {
				selected = options[0]
				return selected, s.atomicSave(s.Path, tmpSelectPrefix, selected)
			}
			return "", nil
		}
		return "", err
	}
	// If the file does exist, try to read as string
	var selection string
	byteData, err := ioutil.ReadAll(file)
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := json.Unmarshal(byteData, &selection); err == nil {
		// file is already migrated, just return selection
		return selection, nil
	}
	// If the file content is not a string, assume we must migrate it
	var configList []Config
	if err := json.Unmarshal(byteData, &configList); err != nil {
		return "", fmt.Errorf("failed to get selected or migrated contexts: %w", err)
	}
	for _, cfg := range configList {
		if err := s.Save(cfg); err != nil {
			return "", fmt.Errorf("failed to migrate context %s: %w", cfg.Name, err)
		}
	}
	// Backup old selection format
	selectBkp := s.Path + ".old"
	if runtime.GOOS == "windows" {
		os.Chmod(selectBkp, 0644) // So that os.Rename works. See https://github.com/golang/go/issues/38287
	}
	if err := os.Rename(s.Path, selectBkp); err != nil {
		return "", fmt.Errorf("failed to backup old contexts: %w", err)
	}
	// And replace it
	var selected string
	if len(configList) > 0 {
		selected = configList[0].Name
	}
	return selected, s.atomicSave(s.Path, tmpSelectPrefix, selected)
}

// List available Configs
func (s *Store) List(ignoreMissing bool) ([]string, error) {
	// Read selected context (the former API contract requires it)
	// also, it makes sure we migrate the contexts, if the user is
	// running an old version.
	if err := s.Read(""); err != nil {
		if !ignoreMissing {
			return nil, err
		}
	}
	return s.listConfigFolder()
}

func (s *Store) listConfigFolder() ([]string, error) {
	dirPath, err := s.getConfigDir()
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(files))
	for _, entry := range files {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		}
	}
	sort.Sort(sort.StringSlice(names))
	return names, nil
}

// Create a named Context
func (s *Store) Create(name string) error {
	if err := s.Save(Config{Name: name}); err != nil {
		return err
	}
	if err := s.Read(name); err != nil {
		return err
	}
	// And update the marker
	return s.atomicSave(s.Path, tmpSelectPrefix, s.Current.Name)
}

// Delete the named config, return the current one
func (s *Store) Delete(name string) error {
	dirPath, err := s.getConfigDir()
	if err != nil {
		return err
	}
	selected, err := s.migrate(dirPath)
	if err != nil {
		return err
	}
	// Remove context
	fullPath := filepath.Join(dirPath, name+".json")
	if err := os.Remove(fullPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	// If the context was selected, replace it by any other
	if selected == name {
		remain, err := s.List(true)
		if err != nil {
			return err
		}
		if len(remain) > 0 {
			selected = remain[0]
		}
	}
	if selected != "" {
		return s.Use(selected) // we must populate s.Current after each call...
	}
	return nil
}

// Use the named Config
func (s *Store) Use(name string) error {
	// Read the context if available
	if err := s.Read(name); err != nil {
		return err
	}
	// And update the marker
	return s.atomicSave(s.Path, tmpSelectPrefix, s.Current.Name)
}

// Info about a particular Config
func (s *Store) Info(name string) (Config, error) {
	var cfg Config
	dirPath, err := s.getConfigDir()
	if err != nil {
		return cfg, err
	}
	// support empty name, meaning whatever context is in use
	if name == "" {
		selection, err := s.migrate(dirPath)
		if err != nil {
			return cfg, err
		}
		if selection == "" {
			return cfg, ErrNoContext
		}
		name = selection
	}
	cfgPath := filepath.Join(dirPath, name+".json")
	file, err := os.Open(cfgPath)
	if err != nil {
		return cfg, fmt.Errorf("context %s could not be read: %w", name, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&cfg); err != nil {
		return cfg, err
	}
	cfg.Name = name
	return cfg, err
}

// Read the config file
func (s *Store) Read(contextName string) error {
	cfg, err := s.Info(contextName)
	if err != nil {
		return err
	}
	s.Current = cfg
	return nil
}

// Dup the current config with a new name
func (s *Store) Dup(name string) error {
	cfg, err := s.Info("")
	if err != nil {
		return err
	}
	dirPath, err := s.getConfigDir()
	if err != nil {
		return err
	}
	// Migrate first
	if _, err := s.migrate(dirPath); err != nil {
		return err
	}
	// Check if the context already exists
	targetPath := filepath.Join(dirPath, name+".json")
	if _, err := os.Stat(targetPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		return fmt.Errorf("target context %s already exists", name)
	}
	// Otherwise, save it
	cfg.Name = name
	if err := s.Save(cfg); err != nil {
		return err
	}
	if err := s.atomicSave(s.Path, tmpSelectPrefix, cfg.Name); err != nil {
		return err
	}
	// And set it as current
	s.Current = cfg
	return nil
}

var aliases = map[string]string{
	"user":    "username",   // alias for username
	"ss":      "subservice", // alias for subservice
	"manager": "iotam",      // alias for iotam
	"cep":     "perseo",     // alias for perseo
}

// CanConfig returns a list of parameter names recognized by `Set`
func (s *Store) CanConfig() []string {
	result := []string{"name"}
	for k := range aliases {
		result = append(result, k)
	}
	for label := range s.Current.pairs() {
		result = append(result, label)
	}
	return result
}

func (s *Store) Set(contextName string, strPairs map[string]string) ([]string, error) {
	updated := make([]string, 0, len(strPairs))
	dirPath, err := s.getConfigDir()
	if err != nil {
		return nil, err
	}
	selectName, err := s.migrate(dirPath)
	if err != nil {
		return nil, err
	}
	cfg, err := s.Info(contextName)
	if err != nil {
		return nil, err
	}
	formerName := ""
	pairs := cfg.pairs()
	for param, value := range strPairs {
		if alias, found := aliases[param]; found {
			param = alias
		}
		switch param {
		case "name":
			if cfg.Name != value {
				_, err = os.Stat(filepath.Join(dirPath, value+".json"))
				if err != nil {
					if !os.IsNotExist(err) {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("context %s already exists", value)
				}
				formerName = cfg.Name
				cfg.Name = value
			}
		case "token":
			if value == HiddenToken {
				value = ""
			}
			cfg.Token = value
		case "urbotoken":
			fallthrough
		case "urboToken":
			if value == HiddenToken {
				value = ""
			}
			cfg.UrboToken = value
		default:
			if v, ok := pairs[param]; ok {
				*v = value
				updated = append(updated, param)
			} else {
				return nil, fmt.Errorf("unknown config parameter %s", param)
			}
		}
	}
	cfg.defaults()
	if err := s.Save(cfg); err != nil {
		return nil, err
	}
	// if renamed, remove older file
	if formerName != "" {
		os.Remove(filepath.Join(dirPath, formerName+".json"))
		if selectName == formerName {
			// If the renamed vertical is the active one, change name
			if err := s.atomicSave(s.Path, tmpSelectPrefix, cfg.Name); err != nil {
				return nil, err
			}
		}
	}
	s.Current = cfg
	return updated, nil
}

func (s *Store) SetParams(contextName string, pairs map[string]string) error {
	cfg, err := s.Info(contextName)
	if err != nil {
		return err
	}
	if len(pairs) > 0 {
		params := cfg.Params
		if params == nil {
			params = make(map[string]string)
		}
		for key, val := range pairs {
			if val == "" {
				delete(params, key)
			} else {
				params[key] = val
			}
		}
		cfg.Params = params
		if err := s.Save(cfg); err != nil {
			return err
		}
	}
	s.Current = cfg
	return nil
}
