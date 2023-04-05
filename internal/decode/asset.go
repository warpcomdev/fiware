package decode

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	"github.com/warpcomdev/fiware"
)

type deployerAsset map[string]map[string]json.RawMessage

var is_text = regexp.MustCompile(`^[a-zA-Z0-9 _\-.:]+$`)

func assetToAttrib(asset map[string]json.RawMessage) []fiware.Attribute {
	attrs := make([]fiware.Attribute, 0, len(asset))
	for name, value := range asset {
		// "TextUnrestricted" for anything that is not regular text or boolean
		attrType := "TextUnrestricted"
		var stringVal string
		var boolVal bool
		if err := json.Unmarshal(value, &boolVal); err == nil {
			attrType = "Boolean"
		} else {
			if err := json.Unmarshal(value, &stringVal); err == nil {
				if is_text.MatchString(stringVal) {
					attrType = "Text"
				}
			}
		}
		attrs = append(attrs, fiware.Attribute{
			Name:  name,
			Type:  attrType,
			Value: value,
		})
	}
	return attrs
}

// Get a list of models from NGSIv2 formatted file
func Asset(filename string) ([]fiware.EntityType, []fiware.Entity) {
	infile, err := SkipBOM(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer infile.Close()
	dec := json.NewDecoder(infile)
	var model deployerAsset
	if err := dec.Decode(&model); err != nil {
		log.Fatalf("Failed to decode asset %s: %v", filename, err)
	}
	entityTypes := make([]fiware.EntityType, 0, len(model))
	for key, asset := range model {
		var (
			entityID   string
			entityType string
		)
		switch key {
		case "deployment":
			entityID = getString(asset, "version")
			entityType = "Vertical"
		case "environment":
			entityID = fmt.Sprintf("%s-%s", getString(asset, "environmentLabel"), getString(asset, "environmentType"))
			entityType = "Environment"
		case "instance":
			entityID = getString(asset, "targetVertical")
			entityType = "VerticalInstance"
		default:
			log.Fatalf("Failed to decode asset %s: unrecognized section %s", filename, key)
		}
		entityTypes = append(entityTypes, fiware.EntityType{
			ID:    entityID,
			Type:  entityType,
			Attrs: assetToAttrib(asset),
		})
	}
	entities := make([]fiware.Entity, 0, len(entityTypes))
	for _, et := range entityTypes {
		attrMap := make(map[string]json.RawMessage)
		for _, attr := range et.Attrs {
			attrMap[attr.Name] = attr.Value
		}
		entities = append(entities, fiware.Entity{
			ID:    et.ID,
			Type:  et.Type,
			Attrs: attrMap,
		})
	}
	return entityTypes, entities
}
