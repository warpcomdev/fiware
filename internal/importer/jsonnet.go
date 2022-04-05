package importer

import (
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/warpcomdev/fiware/internal/serialize"
)

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

type JsonnetSerializer struct {
	serialize.BufferedSerializer
}

func (j *JsonnetSerializer) End() {
	// Prepend matched variables
	if len(j.Matched) > 0 {
		for k, v := range j.Matched {
			if _, err := fmt.Fprintf(j.Original, "local %s = std.extVar(%q); // %q;\n", k, k, v); err != nil {
				j.Err = err
				return
			}
		}
		if _, err := j.Original.WriteString("\n"); err != nil {
			j.Err = err
			return
		}
	}
	j.BufferedSerializer.End()
}
