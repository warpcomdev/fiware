package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/decode"
)

func Load(datafile string, params map[string]string, output interface{}, libPath string) error {
	var (
		jsonStr string
		err     error
	)
	if datafile != "" {
		// Use starlark for .star or .py files
		lowerName := strings.ToLower(datafile)
		switch {
		case strings.HasSuffix(lowerName, ".jsonnet") || strings.HasSuffix(lowerName, ".libsonnet"):
			jsonStr, err = loadJsonnet(datafile, params, libPath)
		case strings.HasSuffix(lowerName, ".star") || strings.HasSuffix(lowerName, ".py"):
			jsonStr, err = loadStarlark(datafile, params, libPath)
		case strings.HasSuffix(lowerName, ".csv"): // support for loading a CSV. Only makes sense to delete entities.
			types, entities := decode.CSV(datafile)
			vertical := fiware.Vertical{EntityTypes: types, Entities: entities}
			if v, ok := output.(*fiware.Vertical); ok {
				*v = vertical
				return nil
			}
			// If target is not a vertical, serialize to decode into it later
			var jsonBytes []byte
			if jsonBytes, err = json.Marshal(vertical); err == nil {
				jsonStr = string(jsonBytes)
			}
		default:
			jsonStr, err = loadCue(datafile, params, libPath)
		}
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
