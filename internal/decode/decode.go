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

func Decode(outfile, verticalName, subserviceName, path string) error {

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
	if strings.HasSuffix(strings.ToLower(path), ".csv") {
		models, instances = CSV(path)
	} else {
		models, instances = NGSI(path)
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
