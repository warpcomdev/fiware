package main

import (
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/template"
)

func render(c *cli.Context, params map[string]string) error {

	datapath, libpath := c.String(dataFlag.Name), c.String(libFlag.Name)
	perEntity := c.String(oncePerEntityFlag.Name)
	outPath := c.String(outputFlag.Name)
	templates := c.Args().Slice()
	return renderTemplate(datapath, libpath, outPath, perEntity, templates, params)
}

// stand-alone render function for use in other commands
func renderTemplate(datapath, libpath, outPath, perEntity string, templates []string, params map[string]string) error {

	manifest, err := importer.Load(datapath, params, libpath)
	if err != nil {
		return err
	}

	// Runs is a map from outputfile to manifest
	runs := make(map[string]fiware.Manifest)
	if perEntity == "" {
		// If only running once, add single entry to map.
		runs[outPath] = manifest
	} else {
		// If running once per entity, coutputFlag is a folder.
		// Use a separate manifest per entityType
		for _, et := range manifest.EntityTypes {
			fullOutPath := filepath.Join(outPath, et.Type+"."+perEntity)
			etManifest := manifest
			etManifest.EntityTypes = []fiware.EntityType{et}
			runs[fullOutPath] = etManifest
		}
	}

	for outPath, manifest := range runs {
		output := outputFile(outPath)
		outFile, err := output.Create()
		if err != nil {
			return err
		}
		defer outFile.Close()
		data, err := template.ManifestForTemplate(manifest, params)
		if err != nil {
			return err
		}
		if err := template.Render(templates, data, outFile); err != nil {
			return err
		}
	}
	return nil
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
