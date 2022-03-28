package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-jsonnet"
)

func Load(datafile string, params map[string]string, output interface{}, libPath string) error {
	var (
		jsonStr string
		err     error
	)
	if datafile != "" {
		// Use starlark for .star or .py files
		lowerName := strings.ToLower(datafile)
		if strings.HasSuffix(lowerName, ".star") || strings.HasSuffix(lowerName, ".py") {
			jsonStr, err = loadStarlark(datafile, params, libPath)
		}
		jsonStr, err = loadJsonnet(datafile, params, libPath)
	}
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("failed to unmarshal file %s: %w", datafile, err)
	}
	return nil
}

// loadJsonnet reads a Jsonnet file with the provided params as std.extVars
func loadJsonnet(datafile string, params map[string]string, pathLib string) (string, error) {
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
		return "", fmt.Errorf("failed to load file %s as jsonnet: %w", datafile, err)
	}
	return jsonStr, nil
}
