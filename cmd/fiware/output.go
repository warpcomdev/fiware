package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/serialize"
)

type outputFile string

type closeWriter interface {
	serialize.Writer
	io.Closer
}

func (output outputFile) Create() (closeWriter, error) {
	if output == "" {
		return struct {
			serialize.Writer
			io.Closer
		}{
			os.Stdout,
			io.NopCloser(os.Stdout),
		}, nil
	}
	outfile, err := os.Create(string(output))
	if err != nil {
		return nil, err
	}
	fmt.Printf("writing output to file %s\n", output)
	return outfile, nil
}

func (output outputFile) Encode(outfile serialize.Writer, vertical *fiware.Vertical, params map[string]string) error {
	var lower = strings.ToLower(string(output))
	var encoder serializerWithSetup
	switch {
	case output != "" && (strings.HasSuffix(lower, ".jsonnet") || strings.HasSuffix(lower, ".libsonnet")):
		encoder = &importer.JsonnetSerializer{}
	case output != "" && (strings.HasSuffix(lower, ".py") || strings.HasSuffix(lower, ".star")):
		ext := filepath.Ext(string(output))
		encoder = &importer.StarlarkSerializer{
			Name: string(output[0 : len(output)-len(ext)]),
		}
	default:
		encoder = &importer.CueSerializer{}
	}
	encoder.Setup(outfile, params)
	encoder.Begin()
	vertical.Serialize(encoder)
	encoder.End()
	return encoder.Error()
}
