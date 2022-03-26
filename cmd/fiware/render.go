package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/template"
)

func render(c *cli.Context, params interface{}) error {
	output := c.String("output")
	var outFile *os.File = os.Stdout
	if output != "" {
		var err error
		outFile, err = os.Create(output)
		if err != nil {
			return err
		}
		defer outFile.Close()
	}
	return template.Render(c.String("data"), c.Args().Slice(), params, outFile)
}
