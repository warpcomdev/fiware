package decode

import (
	"encoding/json"
	"log"

	"github.com/warpcomdev/fiware/models"
)

type ngsiEntity map[string]json.RawMessage
type typedAttribute struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func getString(entity ngsiEntity, field string) string {
	var s string
	if err := json.Unmarshal(entity[field], &s); err != nil {
		log.Fatalf("Failed to decode %s: %v", field, err)
	}
	return s
}

func getAttribute(entity ngsiEntity, field string) models.Attribute {
	var ta typedAttribute
	if err := json.Unmarshal(entity[field], &ta); err != nil {
		log.Fatalf("Failed to decode %s: %v", field, err)
	}
	if ta.Type == "" {
		log.Fatalf("Failed to decode attribute '%s': missing type", field)
	}
	return models.Attribute{
		Name:  field,
		Type:  ta.Type,
		Value: ta.Value,
	}
}

// Get a list of models from NGSIv2 formatted file
func NGSI(filename string) ([]models.EntityType, []models.Entity) {
	infile, err := SkipBOM(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer infile.Close()
	dec := json.NewDecoder(infile)
	var model ngsiEntity
	if err := dec.Decode(&model); err != nil {
		log.Fatalf("Failed to decode model %s: %v", filename, err)
	}
	entityid := getString(model, "id")
	entitytype := getString(model, "type")
	attrs := make([]models.Attribute, 0, len(model))
	attrValues := make(map[string]json.RawMessage)
	for key := range model {
		if key == "id" || key == "type" {
			continue
		}
		current := getAttribute(model, key)
		attrs = append(attrs, current)
		attrValues[current.Name] = current.Value
	}
	modelList := []models.EntityType{{
		ID:    entityid,
		Type:  entitytype,
		Attrs: attrs,
	}}
	entities := []models.Entity{{
		ID:    entityid,
		Type:  entitytype,
		Attrs: attrValues,
	}}
	return modelList, entities
}
