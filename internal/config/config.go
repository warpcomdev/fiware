package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
)

// Config almacena información de conexión a un entorno
type Config struct {
	Name        string            `json:"name"`
	KeystoneURL string            `json:"keystone"`
	OrionURL    string            `json:"orion"`
	IotamURL    string            `json:"iotam"`
	PerseoURL   string            `json:"perseo"`
	Service     string            `json:"service"`
	Subservice  string            `json:"subservice"`
	Username    string            `json:"username"`
	Params      map[string]string `json:"params,omitempty"`
}

// naive function to turn a map into a sequence of string pairs
func map2pairs(m map[string]string) [][2]string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	result := make([][2]string, 0, len(keys))
	for _, k := range keys {
		result = append(result, [2]string{k, m[k]})
	}
	return result
}

func formatPairs(pairs [][2]string, format string, sep string) string {
	result := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, fmt.Sprintf(format, pair[0], pair[1]))
	}
	return strings.Join(result, sep)
}

func formatMap(pairs map[string]string, format string, sep string) string {
	return formatPairs(map2pairs(pairs), format, sep)
}

func (c *Config) String() string {
	pairs := [][2]string{
		{"name", c.Name},
		{"keystone", c.KeystoneURL},
		{"orion", c.OrionURL},
		{"iotam", c.IotamURL},
		{"perseo", c.PerseoURL},
		{"service", c.Service},
		{"subservice", c.Subservice},
		{"username", c.Username},
	}
	settings := formatPairs(pairs, "  %q: %q", ",\n")
	if len(c.Params) > 0 {
		params := strings.Join([]string{
			"  \"params\": {",
			formatMap(c.Params, "    %q: %q", ",\n"),
			"  }",
		}, "\n")
		settings = strings.Join([]string{settings, params}, ",\n")
	}
	result := []string{"{", settings, "}"}
	// add single line too for copy/paste
	pairs = pairs[1:] // skip "name"
	result = append(result, fmt.Sprintf(
		"> fiware context set %s",
		formatPairs(pairs, "%s %q", " ")),
	)
	// And params
	if len(c.Params) > 0 {
		result = append(result, fmt.Sprintf(
			"> fiware context params %s",
			formatMap(c.Params, "%s %q", " ")),
		)
	}
	return strings.Join(result, "\n")
}

// Store can manage several configs
type Store struct {
	Path    string
	Current Config
}

// Read the config file
func (s *Store) Read() error {
	configs, err := s.read()
	if err != nil {
		return err
	}
	if len(configs) > 0 {
		s.Current = configs[0]
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
	return []string{
		"user", // alias for username
		"username",
		"service",
		"ss", // alias for subservice
		"subservice",
		"keystone",
		"orion",
		"manager", // alias for iotam
		"iotam",
		"cep", // alias for perseo
		"perseo",
		"name",
	}
}

func (s *Store) Set(pairs []string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	if len(ctx) <= 0 {
		return ErrNoContext
	}
	selected := ctx[0]
	if len(pairs)%2 != 0 {
		return ErrParametersNumber
	}
	for i := 0; i < len(pairs); i += 2 {
		param, value := pairs[i], pairs[i+1]
		switch param {
		case "user":
			fallthrough
		case "username":
			selected.Username = value
		case "service":
			selected.Service = value
		case "ss":
			fallthrough
		case "subservice":
			selected.Subservice = value
		case "keystone":
			selected.KeystoneURL = value
		case "orion":
			selected.OrionURL = value
		case "manager":
			fallthrough
		case "iotam":
			selected.IotamURL = value
		case "cep":
			fallthrough
		case "perseo":
			selected.PerseoURL = value
		case "name":
			for _, curr := range ctx {
				if curr.Name == value {
					return fmt.Errorf("context %s already exists", value)
				}
			}
			selected.Name = value
		default:
			return fmt.Errorf("unknown config parameter %s", param)
		}
	}
	ctx[0] = selected
	if err := s.save(ctx); err != nil {
		return err
	}
	s.Current = ctx[0]
	return nil
}

func (s *Store) SetParams(pairs []string) error {
	ctx, err := s.read()
	if err != nil {
		return err
	}
	if len(ctx) <= 0 {
		return ErrNoContext
	}
	selected := ctx[0]
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
		ctx[0] = selected
		if err := s.save(ctx); err != nil {
			return err
		}
	}
	s.Current = selected
	return nil
}
