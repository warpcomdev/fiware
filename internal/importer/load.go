package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/decode"
)

func Load(datafile string, params map[string]string, libPath string) (fiware.Manifest, error) {
	var (
		jsonStr  string
		err      error
		manifest fiware.Manifest
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
			return fiware.Manifest{EntityTypes: types, Entities: entities}, nil
		default:
			jsonStr, err = loadCue(datafile, params, libPath)
		}
	}
	if err != nil {
		return manifest, err
	}
	// first try to decode as regular manifest
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		// If that fails, decode as deployer config
		rawDecoder := json.NewDecoder(strings.NewReader(jsonStr))
		rawConfig := deployerConfig{}
		if rawErr := rawDecoder.Decode(&rawConfig); rawErr != nil {
			return manifest, fmt.Errorf("failed to unmarshal file %s: %w, then %w", datafile, err, rawErr)
		}
		manifest = rawConfig.ToManifest()
	}
	return manifest, nil
}
