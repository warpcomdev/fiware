package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
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

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Sort(sort.StringSlice(keys))
	return keys
}

func (c *Config) String() string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	e := json.NewEncoder(w)
	hidden := HiddenToken
	pairs := c.pairs()
	pairs["name"] = &c.Name
	if c.Token != "" {
		pairs["token"] = &hidden
	}
	if c.UrboToken != "" {
		pairs["urboToken"] = &hidden
	}
	w.WriteString("{")
	sep := "\n  \""
	sortedPairs := sortedKeys(pairs)
	for _, label := range sortedPairs {
		value := pairs[label]
		writePair(w, sep, label, "\": ", *value)
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
		writePair(w, " ", label, " ", *value)
	}
	if len(c.Params) > 0 {
		w.WriteString("\n> fiware context params")
		sortedParams := sortedKeys(c.Params)
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

// Store can manage several configs
type Store struct {
	Path    string // It no longer contains full contexts, only context selector.
	DirPath string // his holds the actual contexts now
	Current Config
}

// get the proper paths in the new config model
func (s *Store) getPaths() (selectPath, dirPath string, err error) {
	if s.DirPath != "" {
		return s.Path, s.DirPath, nil
	}
	if s.Path == "" {
		return "", "", errors.New("No path configured in store")
	}
	s.DirPath = strings.TrimSuffix(s.Path, filepath.Ext(s.Path)) + ".d"
	return s.Path, s.DirPath, nil
}

// save some file atomically
func (s *Store) atomicSave(fullPath string, tmpPrefix string, data interface{}) error {
	byteData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	folder, _ := filepath.Split(fullPath)
	if err := os.MkdirAll(folder, 0750); err != nil {
		// Ignore error if the folder exists
		if !os.IsExist(err) {
			return err
		}
	}
	file, err := ioutil.TempFile(folder, tmpPrefix)
	if err != nil {
		return err
	}
	defer os.Remove(file.Name()) // just in case
	_, err = file.Write(byteData)
	closeErr := file.Close() // Close before returning and renaming, for windows.
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if runtime.GOOS == "windows" {
		os.Chmod(fullPath, 0644) // So that os.Rename works. See https://github.com/golang/go/issues/38287
	}
	return os.Rename(file.Name(), fullPath)
}

// Atomically save a config
func (s *Store) atomicSaveConfig(cfg Config) error {
	_, dirPath, err := s.getPaths()
	if err != nil {
		return err
	}
	fullPath := filepath.Join(dirPath, cfg.Name+".json")
	return s.atomicSave(fullPath, "fiware-context", cfg)
}

// Migrate the selection file
func (s *Store) migrate(selectPath, dirPath string) (string, error) {
	file, err := os.Open(selectPath)
	if err != nil {
		// If the file does not exist, return empty selection
		if os.IsNotExist(err) {
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
		if err := s.atomicSaveConfig(cfg); err != nil {
			return "", fmt.Errorf("failed to migrate context %s: %w", cfg.Name, err)
		}
	}
	// Backup old selection format
	selectBkp := selectPath + ".old"
	if runtime.GOOS == "windows" {
		os.Chmod(selectBkp, 0644) // So that os.Rename works. See https://github.com/golang/go/issues/38287
	}
	if err := os.Rename(selectPath, selectBkp); err != nil {
		return "", fmt.Errorf("failed to backup old contexts: %w", err)
	}
	// And replace it
	var selected string
	if len(configList) > 0 {
		selected = configList[0].Name
	}
	return selected, s.atomicSave(selectPath, "fiware-select", selected)
}

// List available Configs
func (s *Store) List() ([]string, error) {
	selectPath, dirPath, err := s.getPaths()
	if err != nil {
		return nil, err
	}
	// Migrate if needed
	if _, err := s.migrate(selectPath, dirPath); err != nil {
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
	return names, nil
}

// Create a named Context
func (s *Store) Create(name string) error {
	if err := s.atomicSaveConfig(Config{Name: name}); err != nil {
		return err
	}
	return s.Read(name)
}

// Delete the named config, return the current one
func (s *Store) Delete(name string) error {
	selectPath, dirPath, err := s.getPaths()
	if err != nil {
		return err
	}
	selected, err := s.migrate(selectPath, dirPath)
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
		remain, err := s.List()
		if err != nil {
			return err
		}
		if len(remain) > 0 {
			selected = remain[0]
		}
	}
	s.Read(selected) // we must populate s.Current after each call...
	return nil
}

// Use the named Config
func (s *Store) Use(name string) error {
	// Read the context if available
	if err := s.Read(name); err != nil {
		return err
	}
	selectPath, _, err := s.getPaths()
	if err != nil {
		return err
	}
	// And update the marker
	return s.atomicSave(selectPath, "fiware-select", s.Current.Name)
}

// Info about a particular Config
func (s *Store) Info(name string) (Config, error) {
	var cfg Config
	selectPath, dirPath, err := s.getPaths()
	if err != nil {
		return cfg, err
	}
	// support empty name, meaning whatever context is in use
	if name == "" {
		selection, err := s.migrate(selectPath, dirPath)
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
	selectPath, dirPath, err := s.getPaths()
	if err != nil {
		return err
	}
	// Migrate first
	if _, err := s.migrate(selectPath, dirPath); err != nil {
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
	if err := s.atomicSaveConfig(cfg); err != nil {
		return err
	}
	if err := s.atomicSave(selectPath, "fiware-select", cfg.Name); err != nil {
		return err
	}
	// And set it as current
	s.Current = cfg
	return nil
}

// CanConfig returns a list of parameter names recognized by `Set`
func (s *Store) CanConfig() []string {
	result := []string{
		"name",
		"user",    // alias for username
		"ss",      // alias for subservice
		"manager", // alias for iotam
		"cep",     // alias for perseo
	}
	for label := range s.Current.pairs() {
		result = append(result, label)
	}
	return result
}

func (s *Store) Set(contextName string, strPairs []string) error {
	selectPath, dirPath, err := s.getPaths()
	if err != nil {
		return err
	}
	selectName, err := s.migrate(selectPath, dirPath)
	if err != nil {
		return err
	}
	cfg, err := s.Info(contextName)
	if err != nil {
		return err
	}
	if len(strPairs)%2 != 0 {
		return ErrParametersNumber
	}
	formerName := ""
	pairs := cfg.pairs()
	for i := 0; i < len(strPairs); i += 2 {
		param, value := strPairs[i], strPairs[i+1]
		switch param {
		// aliases
		case "user":
			cfg.Username = value
		case "ss":
			cfg.Subservice = value
		case "manager":
			cfg.IotamURL = value
		case "cep":
			cfg.PerseoURL = value
		case "name":
			if cfg.Name != value {
				_, err = os.Stat(filepath.Join(dirPath, value+".json"))
				if err != nil {
					if !os.IsNotExist(err) {
						return err
					}
				} else {
					return fmt.Errorf("context %s already exists", value)
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
			} else {
				return fmt.Errorf("unknown config parameter %s", param)
			}
		}
	}
	cfg.defaults()
	if err := s.atomicSaveConfig(cfg); err != nil {
		return err
	}
	// if renamed, remove older file
	if formerName != "" {
		os.Remove(filepath.Join(dirPath, formerName+".json"))
		if selectName == formerName {
			// If the renamed vertical is the active one, change name
			if err := s.atomicSave(selectPath, "fiware-select", cfg.Name); err != nil {
				return err
			}
		}
	}
	s.Current = cfg
	return nil
}

func (s *Store) SetParams(contextName string, pairs []string) error {
	cfg, err := s.Info(contextName)
	if err != nil {
		return err
	}
	if len(pairs)%2 != 0 {
		return ErrParametersNumber
	}
	if len(pairs) > 0 {
		params := cfg.Params
		if params == nil {
			params = make(map[string]string)
		}
		for i := 0; i < len(pairs); i += 2 {
			key, val := pairs[i], pairs[i+1]
			if val == "" {
				delete(params, key)
			} else {
				params[key] = val
			}
		}
		cfg.Params = params
		if err := s.atomicSaveConfig(cfg); err != nil {
			return err
		}
	}
	s.Current = cfg
	return nil
}
