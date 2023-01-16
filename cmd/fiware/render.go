package main

import (
	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/template"
)

func render(c *cli.Context, params map[string]string) error {
	output := outputFile(c.String(outputFlag.Name))
	outFile, err := output.Create()
	if err != nil {
		return err
	}
	defer outFile.Close()

	datapath, libpath := c.String(dataFlag.Name), c.String(libFlag.Name)
	manifest, err := importer.Load(datapath, params, libpath)
	if err != nil {
		return err
	}
	data, err := template.ManifestForTemplate(manifest, params)
	if err != nil {
		return err
	}
	return template.Render(c.Args().Slice(), data, outFile)
}

func export(c *cli.Context, params map[string]string) error {
	output := outputFile(c.String(outputFlag.Name))
	outFile, err := output.Create()
	if err != nil {
		return err
	}
	defer outFile.Close()

	datapath, libpath := c.String(dataFlag.Name), c.String(libFlag.Name)
	manifest, err := importer.Load(datapath, params, libpath)
	if err != nil {
		return err
	}
	return output.Encode(outFile, &manifest, params)
}
