package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/template"
)

type verticalWithParams struct {
	fiware.Vertical
	Params map[string]string `json:"params,omitempty"`
}

func render(c *cli.Context, params map[string]string) error {
	output := c.String(outputFlag.Name)
	var outFile *os.File = os.Stdout
	if output != "" {
		var err error
		outFile, err = os.Create(output)
		if err != nil {
			return err
		}
		defer outFile.Close()
	}

	datapath, libpath := c.String(dataFlag.Name), c.String(libFlag.Name)
	var data verticalWithParams
	if err := importer.Load(datapath, params, &data.Vertical, libpath); err != nil {
		return err
	}
	data.Params = params
	return template.Render(c.Args().Slice(), data, outFile)
}
