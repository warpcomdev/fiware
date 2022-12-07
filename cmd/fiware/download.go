package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
)

func listVerticals(c *cli.Context, store *config.Store) ([]string, error) {
	selected, err := getConfig(c, store)
	if err != nil {
		return nil, err
	}
	vertical := &fiware.Manifest{Subservice: selected.Subservice}
	client := httpClient(c.Bool(verboseFlag.Name))
	u, header, err := getUrboHeaders(c, selected)
	if err != nil {
		return nil, err
	}
	if err := getVerticals(selected, client, u, header, vertical); err != nil {
		return nil, err
	}
	v := make([]string, 0, len(vertical.Verticals))
	for _, item := range vertical.Verticals {
		v = append(v, item.Slug)
	}
	return v, nil
}

func ensureDir(outdir string) error {
	stat, err := os.Stat(outdir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("Failed to check folder %s: %w", outdir, err)
		}
		if err := os.MkdirAll(outdir, 0750); err != nil {
			return fmt.Errorf("Failed to create folder %s: %w", outdir, err)
		}
	} else {
		if !stat.IsDir() {
			return fmt.Errorf("path %s already exists and it is not a directory", outdir)
		}
	}
	return nil
}

func downloadResource(c *cli.Context, store *config.Store) error {

	if c.NArg() <= 0 {
		v, err := listVerticals(c, store)
		if err != nil {
			return fmt.Errorf("select a resource from: %s", strings.Join(v, ", "))
		}
		return errors.New("login first and then select a vertical")
	}

	selected, err := getConfig(c, store)
	if err != nil {
		return err
	}

	outdir := c.String(outdirFlag.Name)
	if err := ensureDir(outdir); err != nil {
		return err
	}

	vertical := &fiware.Manifest{Subservice: selected.Subservice}
	client := httpClient(c.Bool(verboseFlag.Name))
	u, header, err := getUrboHeaders(c, selected)
	if err != nil {
		return err
	}
	if err := getVerticals(selected, client, u, header, vertical); err != nil {
		return err
	}

	output := outputFile(filepath.Join(outdir, "00_verticals.json"))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()
	manifest := fiware.Manifest{
		Deployment: fiware.DeploymentManifest{
			Sources: make(map[string]fiware.ManifestSource),
		},
	}

	for _, target := range c.Args().Slice() {
		match := false
		for _, v := range vertical.Verticals {
			if v.Slug == target {
				// Output is saved in manifest format
				clean_vertical := v
				clean_vertical.UrboVerticalStatus = fiware.UrboVerticalStatus{}
				slugManifest := fiware.Manifest{
					Verticals: map[string]fiware.Vertical{
						v.Slug: clean_vertical,
					},
					ManifestPanels: fiware.PanelManifest{
						Sources: make(map[string]fiware.ManifestSource),
					},
				}
				// And save the panel list too
				paneldir := filepath.Join(outdir, v.Slug)
				if err := ensureDir(paneldir); err != nil {
					return err
				}
				sources := fiware.ManifestSource{
					Path:  v.Slug,
					Files: make([]string, 0, len(v.Panels)+len(v.ShadowPanels)),
				}
				for _, panel := range v.Panels {
					fileName, err := downloadPanel(u, client, header, panel, paneldir)
					if err != nil {
						return err
					}
					sources.Files = append(sources.Files, fileName)
				}
				for _, panel := range v.ShadowPanels {
					fileName, err := downloadPanel(u, client, header, panel, paneldir)
					if err != nil {
						return err
					}
					sources.Files = append(sources.Files, fileName)
				}
				match = true
				slugFilename := fmt.Sprintf("%s.json", v.Slug)
				slugOutput := outputFile(filepath.Join(outdir, slugFilename))
				slugOutfile, err := slugOutput.Create()
				if err != nil {
					return err
				}
				slugOutput.Encode(slugOutfile, slugManifest, nil)
				slugOutfile.Close()
				manifest.Deployment.Sources["vertical:"+v.Slug] = fiware.ManifestSource{
					Files: []string{slugFilename},
				}
			}
		}
		if !match {
			return fmt.Errorf("vertical %s not found", target)
		}
	}
	return output.Encode(outfile, manifest, nil)
}

func downloadPanel(u *urbo.Urbo, client keystone.HTTPClient, header http.Header, slug string, outdir string) (string, error) {
	data, err := u.DownloadPanel(client, header, slug)
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%s.json", slug)
	output := outputFile(filepath.Join(outdir, fileName))
	outfile, err := output.Create()
	if err != nil {
		return "", err
	}
	defer outfile.Close()
	if _, err := outfile.Write(data); err != nil {
		return "", err
	}
	return fileName, nil
}
