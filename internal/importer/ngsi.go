package importer

import (
	"encoding/json"
	"fmt"

	"github.com/warpcomdev/fiware"
)

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
		attrType, ok := attrib["type"]
		if !ok {
			return fiware.Manifest{}, fmt.Errorf("Attribute %s has no type", k)
		}
		if err := json.Unmarshal(attrType, &attrTypeStr); err != nil {
			return fiware.Manifest{}, fmt.Errorf("Failed to decode type of attribute %s: %w", k, err)
		}
		entityType.Attrs = append(entityType.Attrs, fiware.Attribute{
			Name: k,
			Type: attrTypeStr,
		})
		attrValue, ok := attrib["value"]
		if !ok {
			return fiware.Manifest{}, fmt.Errorf("Attribute %s has no value", k)
		}
		entity.Attrs[k] = attrValue
	}
	manifest := fiware.Manifest{
		EntityTypes: []fiware.EntityType{entityType},
		Entities:    []fiware.Entity{entity},
	}
	return manifest, nil
}
