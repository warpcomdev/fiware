package main

import (
	"encoding/json"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/importer"
	"github.com/warpcomdev/fiware/internal/template"
)

type verticalWithParams struct {
	fiware.Manifest
	Params map[string]string `json:"params,omitempty"`
}

func render(c *cli.Context, params map[string]string) error {
	output := outputFile(c.String(outputFlag.Name))
	outFile, err := output.Create()
	if err != nil {
		return err
	}
	defer outFile.Close()

	datapath, libpath := c.String(dataFlag.Name), c.String(libFlag.Name)
	var data interface{}
	if c.Bool(relaxedFlag.Name) {
		var relaxedData map[string]interface{}
		if err := importer.Load(datapath, params, &relaxedData, libpath); err != nil {
			return err
		}
		if len(params) > 0 {
			relaxedData["params"] = params
		}
		data = relaxedData
	} else {
		var strictData verticalWithParams
		if err := importer.Load(datapath, params, &strictData.Manifest, libpath); err != nil {
			return err
		}
		strictData.Params = params
		// Convierto a map[string]interface{} pasando por json,
		// porque no quiero que los diseñadores de los templates
		// necesiten conocer el formato de los objetos golang.
		// Mejor que puedan trabajar con la misma estructura de atributos
		// que en el fichero de datos.
		text, err := json.Marshal(strictData)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(text, &data); err != nil {
			return err
		}
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
	var vertical fiware.Manifest
	if err := importer.Load(datapath, params, &vertical, libpath); err != nil {
		return err
	}
	return output.Encode(outFile, &vertical, params)
}
