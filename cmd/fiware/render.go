package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/template"
)

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
	var data interface{}
	if err := importer.Load(datapath, params, &data, libpath); err != nil {
		return err
	}
	return template.Render(c.Args().Slice(), data, params, outFile)
}
