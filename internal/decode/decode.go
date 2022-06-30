package decode

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/warpcomdev/fiware"
)

//go:embed vertical.cue
var verticalTemplate string

const (
	fromMarker = "// BEGIN REPLACE"
	toMarker   = "// END REPLACE"
)

func Decode(outfile, verticalName, subserviceName string, paths []string) error {

	fromIndex := strings.Index(verticalTemplate, fromMarker)
	toIndex := strings.Index(verticalTemplate, toMarker)
	if fromIndex < 0 || toIndex < 0 {
		return errors.New("failed to replace markers in verticals input file")
	}

	var handle *os.File = os.Stdout
	if outfile != "" {
		var err error
		if handle, err = os.Create(outfile); err != nil {
			return fmt.Errorf("failed to open output file %s: %w", outfile, err)
		}
		defer handle.Close()
	}

	var (
		models    []fiware.EntityType
		instances []fiware.Entity
	)
	// Allow reading both a NGSI and a CSV file. If both specified,
	// entity types are read from NGSI file, but entity values are
	// read from CSV.
	for _, path := range paths {
		if strings.HasSuffix(strings.ToLower(path), ".csv") {
			localModels, localInstances := CSV(path)
			instances = localInstances
			if models == nil {
				models = localModels
			}
		} else {
			localModels, localInstances := NGSI(path)
			models = localModels
			if instances == nil {
				instances = localInstances
			}
		}
	}
	indent := "    "
	modelText, err := json.MarshalIndent(models, indent, indent)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}
	var instanceText []byte
	if len(instances) > 0 {
		instanceText, err = json.MarshalIndent(instances, indent, indent)
		if err != nil {
			return fmt.Errorf("failed to marshal instances: %w", err)
		}
	}

	handle.WriteString(verticalTemplate[:fromIndex])
	handle.WriteString(fmt.Sprintf(
		"\n%s\"name\": %q,\n%s\"subservice\": %q,\n%s\"entityTypes\": ",
		indent, verticalName, indent, subserviceName, indent,
	))
	handle.Write(modelText)
	if len(instanceText) > 0 {
		handle.WriteString(fmt.Sprintf(",\n%s\"entities\": ", indent))
		handle.Write(instanceText)
	}
	handle.WriteString(",\n")
	handle.WriteString(verticalTemplate[toIndex+len(toMarker):])
	handle.WriteString("\n")
	return nil
}
