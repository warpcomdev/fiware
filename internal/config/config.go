package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
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
	for label, value := range pairs {
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
	for label, value := range pairs {
		writePair(w, " ", label, " ", *value)
	}
	if len(c.Params) > 0 {
		w.WriteString("\n> fiware context params")
		for k, v := range c.Params {
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
	Path    string
	Current Config
}

// Read the config file
func (s *Store) Read(contextName string) error {
	configs, err := s.read()
	if err != nil {
		return err
	}
	if len(configs) > 0 {
		selected, err := findByName(configs, contextName)
		if err != nil {
			return err
		}
		s.Current = configs[selected]
	}
	return nil
}

func (s *Store) read() ([]Config, error) {
	infile, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Config{}, nil
		}
		return nil, err
	}
	defer infile.Close()
	decoder := json.NewDecoder(infile)
	var result []Config
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// save the config file
func (s Store) save(c []Config) error {
	file, err := ioutil.TempFile(path.Dir(s.Path), "fiware")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	encoder := json.NewEncoder(file)
	err = encoder.Encode(c)
	file.Close() // Close before returning an renaming, for windows.
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		os.Chmod(s.Path, 0644) // So that os.Rename works. See https://github.com/golang/go/issues/38287
	}
	return os.Rename(file.Name(), s.Path)
}

// Create a named Cotext
func (s *Store) Create(name string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	for _, curr := range ctx {
		if curr.Name == name {
			return fmt.Errorf("config %s already exists", name)
		}
	}
	newCtx := make([]Config, 0, len(ctx)+1)
	newCtx = append(append(newCtx, Config{Name: name}), ctx...)
	if err := s.save(newCtx); err != nil {
		return err
	}
	s.Current = newCtx[0]
	return nil
}

// Delete the named config, return the current one
func (s *Store) Delete(name string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	for index, curr := range ctx {
		if curr.Name == name {
			ctx = append(ctx[:index], ctx[index+1:]...)
			break
		}
	}
	if err := s.save(ctx); err != nil {
		return err
	}
	if len(ctx) > 0 {
		s.Current = ctx[0]
	}
	return nil
}

// List available Configs
func (s *Store) List() ([]string, error) {
	ctx, err := s.read()
	if err != nil {
		return nil, err
	}
	if len(ctx) <= 0 {
		return nil, nil
	}
	result := make([]string, 0, len(ctx))
	for _, curr := range ctx {
		result = append(result, curr.Name)
	}
	return result, nil
}

// Use the named Config
func (s *Store) Use(name string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	if name != "" {
		for index, curr := range ctx {
			if curr.Name == name {
				if index > 0 {
					ctx = append(append([]Config{ctx[index]}, ctx[:index]...), ctx[index+1:]...)
					if err := s.save(ctx); err != nil {
						return err
					}
				}
				break
			}
		}
	}
	if len(ctx) > 0 {
		s.Current = ctx[0]
		s.Current.defaults()
	}
	return nil
}

// Info about a particular Config
func (s *Store) Info(name string) (Config, error) {
	ctx, err := s.read()
	if err != nil {
		return Config{}, err
	}
	var selectedIndex int
	if name != "" {
		for index, curr := range ctx {
			if curr.Name == name {
				selectedIndex = index
				break
			}
		}
	}
	if len(ctx) <= selectedIndex {
		return Config{}, nil
	}
	return ctx[selectedIndex], nil
}

// Dup the current config with a new name
func (s *Store) Dup(name string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	if len(ctx) <= 0 {
		return ErrNoContext
	}
	newCtx := ctx[0]
	newCtx.Name = name
	if err := s.save(append([]Config{newCtx}, ctx...)); err != nil {
		return err
	}
	s.Current = newCtx
	return nil
}

// CanConfig returns a list of parameter names recoginzed by `Set`
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
	ctx, err := s.read()
	if err != nil {
		return err
	}
	if len(ctx) <= 0 {
		return ErrNoContext
	}
	selectedIndex, err := findByName(ctx, contextName)
	if err != nil {
		return err
	}
	selected := ctx[selectedIndex]
	if len(strPairs)%2 != 0 {
		return ErrParametersNumber
	}
	pairs := selected.pairs()
	for i := 0; i < len(strPairs); i += 2 {
		param, value := strPairs[i], strPairs[i+1]
		switch param {
		// aliases
		case "user":
			selected.Username = value
		case "ss":
			selected.Subservice = value
		case "manager":
			selected.IotamURL = value
		case "cep":
			selected.PerseoURL = value
		case "name":
			for _, curr := range ctx {
				if curr.Name == value {
					return fmt.Errorf("context %s already exists", value)
				}
			}
			selected.Name = value
		case "token":
			if value == HiddenToken {
				value = ""
			}
			selected.Token = value
		case "urbotoken":
			fallthrough
		case "urboToken":
			if value == HiddenToken {
				value = ""
			}
			selected.UrboToken = value
		default:
			if v, ok := pairs[param]; ok {
				*v = value
			} else {
				return fmt.Errorf("unknown config parameter %s", param)
			}
		}
	}
	selected.defaults()
	ctx[selectedIndex] = selected
	if err := s.save(ctx); err != nil {
		return err
	}
	s.Current = ctx[selectedIndex]
	return nil
}

func (s *Store) SetParams(contextName string, pairs []string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	if len(ctx) <= 0 {
		return ErrNoContext
	}
	selectedIndex, err := findByName(ctx, contextName)
	if err != nil {
		return err
	}
	selected := ctx[selectedIndex]
	if len(pairs)%2 != 0 {
		return ErrParametersNumber
	}
	if len(pairs) > 0 {
		params := selected.Params
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
		selected.Params = params
		ctx[selectedIndex] = selected
		if err := s.save(ctx); err != nil {
			return err
		}
	}
	s.Current = selected
	return nil
}

func findByName(ctx []Config, contextName string) (int, error) {
	if contextName == "" {
		return 0, nil
	}
	for index, current := range ctx {
		if current.Name == contextName {
			return index, nil
		}
	}
	return -1, fmt.Errorf("context %s not found", contextName)
}
