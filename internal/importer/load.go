package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
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
		// If that fails, decode as NGSI entity
		ngsiManifest, ngsiErr := decode_ngsi(jsonStr)
		if ngsiErr == nil {
			manifest = ngsiManifest
		} else {
			// Finally, try to decode as deployer config
			rawDecoder := json.NewDecoder(strings.NewReader(jsonStr))
			rawConfig := deployerConfig{}
			if rawErr := rawDecoder.Decode(&rawConfig); rawErr != nil {
				return manifest, fmt.Errorf("failed to unmarshal file %s: %w, then %w and finally %w", datafile, err, rawErr, ngsiErr)
			}
			manifest = rawConfig.ToManifest()
		}
	}
	// Always add notification endpoints
	if manifest.Environment.NotificationEndpoints == nil {
		manifest.Environment.NotificationEndpoints = make(map[string]string)
	}
	for key, val := range config.EndpointsFromParams(params) {
		manifest.Environment.NotificationEndpoints[key] = val
	}
	return manifest, nil
}

func decode_ngsi(jsonStr string) (fiware.Manifest, error) {
	var data map[string]json.RawMessage
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return fiware.Manifest{}, err
	}
	eId, hasId := data["id"]
	eType, hasType := data["type"]
	if !hasId || !hasType {
		return fiware.Manifest{}, fmt.Errorf("missing id or type in NGSI entity")
	}
	var idStr, typeStr string
	if err := json.Unmarshal(eId, &idStr); err != nil {
		return fiware.Manifest{}, fmt.Errorf("failed to decode id: %w", err)
	}
	if err := json.Unmarshal(eType, &typeStr); err != nil {
		return fiware.Manifest{}, fmt.Errorf("failed to decode type: %w", err)
	}
	entityType := fiware.EntityType{
		ID:    idStr,
		Type:  typeStr,
		Attrs: make([]fiware.Attribute, 0, len(data)),
	}
	entity := fiware.Entity{
		ID:        idStr,
		Type:      typeStr,
		Attrs:     make(map[string]json.RawMessage),
		MetaDatas: make(map[string]json.RawMessage),
	}
	for k, v := range data {
		if k == "id" || k == "type" {
			continue
		}
		var attrib map[string]json.RawMessage
		if err := json.Unmarshal(v, &attrib); err != nil {
			return fiware.Manifest{}, fmt.Errorf("Failed to decode attribute %s: %w", k, err)
		}
		var attrTypeStr string
		if err := json.Unmarshal(attrib["type"], &attrTypeStr); err != nil {
			return fiware.Manifest{}, fmt.Errorf("Failed to decode type of attribute %s: %w", k, err)
		}
		entityType.Attrs = append(entityType.Attrs, fiware.Attribute{
			Name: k,
			Type: attrTypeStr,
		})
		entity.Attrs[k] = v
	}
	manifest := fiware.Manifest{
		EntityTypes: []fiware.EntityType{entityType},
		Entities:    []fiware.Entity{entity},
	}
	return manifest, nil
}
