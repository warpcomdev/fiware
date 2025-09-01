package decode

import (
	"encoding/json"
	"log"

	"github.com/warpcomdev/fiware/models"
)

type builderModel map[string]struct {
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
	ExampleId   string `json:"exampleId"`
	Model       map[string]struct {
		NGSIType    string          `json:"ngsiType"`
		DBType      string          `json:"dbType"`
		Description string          `json:"description"`
		Extra       string          `json:"extra"`
		Unit        string          `json:"unit"`
		Range       string          `json:"range"`
		Example     json.RawMessage `json:"example"`
	} `json:"model"`
}

// Get a list of models from Builder file.
func Builder(filename string) ([]models.EntityType, []models.Entity) {
	infile, err := SkipBOM(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer infile.Close()
	dec := json.NewDecoder(infile)
	var model builderModel
	if err := dec.Decode(&model); err != nil {
		log.Fatalf("Failed to decode model %s: %v", filename, err)
	}
	modelList := make([]models.EntityType, 0, len(model))
	for modelType, modelData := range model {
		attrs := make([]models.Attribute, 0, len(modelData.Model))
		for label, attrData := range modelData.Model {
			attr := models.Attribute{
				Name: label,
				Type: attrData.NGSIType,
				Description: []string{
					attrData.Description,
					attrData.Extra,
					attrData.Unit,
					attrData.Range,
				},
				Value: attrData.Example,
			}
			attrs = append(attrs, attr)
		}
		et := models.EntityType{
			ID:    modelData.ExampleId,
			Type:  modelType,
			Attrs: attrs,
		}
		modelList = append(modelList, et)
	}
	return modelList, nil
}
