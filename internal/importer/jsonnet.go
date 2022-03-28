package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-jsonnet"
)

func Load(datafile string, params map[string]string, output interface{}, libPath string) error {
	if datafile != "" {
		// Use starlark for .star or .py files
		lowerName := strings.ToLower(datafile)
		if strings.HasSuffix(lowerName, ".star") || strings.HasSuffix(lowerName, ".py") {
			return LoadStarlark(datafile, params, output, libPath)
		}
		return LoadJsonnet(datafile, params, output, libPath)
	}
	return nil
}

// LoadJsonnet reads a Jsonnet file with the provided params as std.extVars
func LoadJsonnet(datafile string, params map[string]string, output interface{}, pathLib string) error {
	vm := jsonnet.MakeVM()
	for k, v := range params {
		vm.ExtVar(k, v)
	}
	if pathLib != "" {
		vm.Importer(&jsonnet.FileImporter{
			JPaths: []string{pathLib},
		})
	}
	jsonStr, err := vm.EvaluateFile(datafile)
	if err != nil {
		return fmt.Errorf("failed to load file %s as jsonnet: %w", datafile, err)
	}
	if err := json.Unmarshal([]byte(jsonStr), output); err != nil {
		return fmt.Errorf("failed to unmarshal file %s as json: %w", datafile, err)
	}
	return nil
}
