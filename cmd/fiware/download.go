package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/serialize"
	"github.com/warpcomdev/fiware/internal/urbo"
)

type verticalDownloader struct {
	Selected      config.Config
	Manifest      fiware.Manifest
	Client        keystone.HTTPClient
	Urbo          *urbo.Urbo
	Headers       http.Header
	VerticalNames []string
}

func newVerticalDownloader(c *cli.Context, store *config.Store) (*verticalDownloader, error) {
	var (
		downloader verticalDownloader
		err        error
	)
	downloader.Selected, err = getConfig(c, store)
	if err != nil {
		return nil, err
	}
	downloader.Manifest.Subservice = downloader.Selected.Subservice
	downloader.Client = httpClient(c.Bool(verboseFlag.Name))
	downloader.Urbo, downloader.Headers, err = getUrboHeaders(c, downloader.Selected)
	if err != nil {
		return nil, err
	}
	if err := getVerticals(downloader.Selected, downloader.Client, downloader.Urbo, downloader.Headers, &downloader.Manifest); err != nil {
		return nil, err
	}
	downloader.VerticalNames = make([]string, 0, len(downloader.Manifest.Verticals))
	for _, item := range downloader.Manifest.Verticals {
		downloader.VerticalNames = append(downloader.VerticalNames, item.Slug)
	}
	return &downloader, nil
}

// Dowload all panels in vertical, return file name of manifest written
func (d *verticalDownloader) Download(v fiware.Vertical, outdir string) (string, error) {
	clean_vertical := v
	clean_vertical.UrboVerticalStatus = fiware.UrboVerticalStatus{}
	manifest := fiware.Manifest{
		Verticals: map[string]fiware.Vertical{
			v.Slug: clean_vertical,
		},
		ManifestPanels: fiware.PanelManifest{
			Sources: make(map[string]fiware.ManifestSource),
		},
	}
	paneldir := filepath.Join(outdir, v.Slug)
	if err := ensureDir(paneldir); err != nil {
		return "", err
	}
	sources := fiware.ManifestSource{
		Path:  v.Slug,
		Files: make([]string, 0, len(v.Panels)+len(v.ShadowPanels)),
	}
	for _, panel := range v.Panels {
		fileName, err := downloadPanel(d.Urbo, d.Client, d.Headers, panel, paneldir)
		if err != nil {
			return "", err
		}
		sources.Files = append(sources.Files, fileName)
	}
	for _, panel := range v.ShadowPanels {
		fileName, err := downloadPanel(d.Urbo, d.Client, d.Headers, panel, paneldir)
		if err != nil {
			return "", err
		}
		sources.Files = append(sources.Files, fileName)
	}
	manifestFilename := fmt.Sprintf("%s.json", v.Slug)
	manifestFullname := outputFile(filepath.Join(outdir, manifestFilename))
	manifestFile, err := manifestFullname.Create()
	if err != nil {
		return "", err
	}
	defer manifestFile.Close()
	manifestFullname.Encode(manifestFile, manifest, nil)
	return manifestFilename, nil
}

func listVerticals(c *cli.Context, store *config.Store) ([]string, error) {
	downloader, err := newVerticalDownloader(c, store)
	if err != nil {
		return nil, err
	}
	return downloader.VerticalNames, nil
}

type projectDownloader struct {
	Selected     config.Config
	Manifest     fiware.Manifest
	Client       keystone.HTTPClient
	Keystone     *keystone.Keystone
	Headers      http.Header
	ProjectNames []string
}

func newProjectDownloader(c *cli.Context, store *config.Store) (*projectDownloader, error) {
	var (
		downloader projectDownloader
		err        error
	)
	downloader.Selected, err = getConfig(c, store)
	if err != nil {
		return nil, err
	}
	downloader.Manifest.Subservice = downloader.Selected.Subservice
	downloader.Client = httpClient(c.Bool(verboseFlag.Name))
	downloader.Keystone, downloader.Headers, err = getKeystoneHeaders(c, downloader.Selected)
	if err != nil {
		return nil, err
	}
	if err := getProjects(downloader.Selected, downloader.Client, downloader.Keystone, downloader.Headers, &downloader.Manifest); err != nil {
		return nil, err
	}
	downloader.ProjectNames = make([]string, 0, len(downloader.Manifest.Verticals))
	for _, item := range downloader.Manifest.Projects {
		downloader.ProjectNames = append(downloader.ProjectNames, item.Name)
	}
	return &downloader, nil
}

func listProjects(c *cli.Context, store *config.Store, manifest *fiware.Manifest) ([]string, error) {
	downloader, err := newProjectDownloader(c, store)
	if err != nil {
		return nil, err
	}
	return downloader.ProjectNames, nil
}

// Dowload all panels in vertical, return file name of manifest written
func (d *projectDownloader) Download(v fiware.Project, outdir string) (string, error) {
	trimmedSubservice := strings.TrimLeft(v.Name, "/")
	manifestFilename := fmt.Sprintf("%s.json", trimmedSubservice)
	manifestFullname := outputFile(filepath.Join(outdir, manifestFilename))
	manifestFile, err := manifestFullname.Create()
	if err != nil {
		return "", err
	}
	defer manifestFile.Close()
	assetdir := filepath.Join(outdir, trimmedSubservice)
	if err := ensureDir(assetdir); err != nil {
		return "", err
	}
	resources := map[string]func(config.Config, keystone.HTTPClient, http.Header, *fiware.Manifest) error{
		"rules":         getRules,
		"subscriptions": getSuscriptions,
		"groups":        getServices,
		"devices":       getDevices,
	}
	assetSource := fiware.ManifestSource{
		Path:  trimmedSubservice,
		Files: make([]string, 0, len(resources)),
	}
	for label, getter := range resources {
		var assetManifest fiware.Manifest
		if err := getter(d.Selected, d.Client, d.Headers, &assetManifest); err != nil {
			return "", err
		}
		assetManifest.ClearStatus()
		// Save the resources in a separate manifest file
		assetFilename := fmt.Sprintf("%s.json", label)
		assetFullname := outputFile(filepath.Join(outdir, trimmedSubservice, assetFilename))
		assetFile, err := assetFullname.Create()
		if err != nil {
			return "", err
		}
		assetFullname.Encode(assetFile, assetManifest, nil)
		assetFile.Close()
		// And add the manifest file as a source
		assetSource.Files = append(assetSource.Files, assetFilename)
	}
	manifest := fiware.Manifest{
		Deployment: fiware.DeploymentManifest{
			Sources: map[string]fiware.ManifestSource{
				fmt.Sprintf("subservice:%s:assets", trimmedSubservice): assetSource,
			},
		},
	}
	manifestFullname.Encode(manifestFile, manifest, nil)
	return manifestFilename, nil
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
	downloader, err := newVerticalDownloader(c, store)
	if err != nil {
		return err
	}

	allTargets := c.Bool(allFlag.Name)
	if c.NArg() <= 0 && !allTargets {
		return fmt.Errorf("select a resource from: %s", strings.Join(downloader.VerticalNames, ", "))
	}
	targetSlugs := make(map[string]bool)
	for _, target := range c.Args().Slice() {
		targetSlugs[target] = false
	}

	manifest := fiware.Manifest{
		Deployment: fiware.DeploymentManifest{
			Sources: make(map[string]fiware.ManifestSource),
		},
	}

	outdir := c.String(outdirFlag.Name)
	if err := ensureDir(outdir); err != nil {
		return err
	}

	output := outputFile(filepath.Join(outdir, "00_verticals.json"))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()

	for _, v := range downloader.Manifest.Verticals {
		if _, ok := targetSlugs[v.Slug]; !ok && !allTargets {
			continue
		}
		targetSlugs[v.Slug] = true
		// Output is saved in manifest format
		filename, err := downloader.Download(v, outdir)
		if err != nil {
			return err
		}
		manifest.Deployment.Sources["vertical:"+v.Slug] = fiware.ManifestSource{
			Files: []string{filename},
		}
	}

	return checkTargets(targetSlugs, output, outfile, manifest)
}

func downloadProject(c *cli.Context, store *config.Store) error {
	downloader, err := newProjectDownloader(c, store)
	if err != nil {
		return err
	}

	allTargets := c.Bool(allFlag.Name)
	if c.NArg() <= 0 && !allTargets {
		return fmt.Errorf("select a resource from: %s", strings.Join(downloader.ProjectNames, ", "))
	}
	targetNames := make(map[string]bool)
	for _, target := range c.Args().Slice() {
		if !strings.HasPrefix(target, "/") {
			target = fmt.Sprintf("/%s", target)
		}
		targetNames[target] = false
	}

	manifest := fiware.Manifest{
		Deployment: fiware.DeploymentManifest{
			Sources: make(map[string]fiware.ManifestSource),
		},
	}

	outdir := c.String(outdirFlag.Name)
	if err := ensureDir(outdir); err != nil {
		return err
	}

	output := outputFile(filepath.Join(outdir, "00_assets.json"))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()

	for _, v := range downloader.Manifest.Projects {
		if _, ok := targetNames[v.Name]; !ok && !allTargets {
			continue
		}
		targetNames[v.Name] = true
		// Output is saved in manifest format
		filename, err := downloader.Download(v, outdir)
		if err != nil {
			return err
		}
		manifest.Deployment.Sources["subservice:"+v.Name] = fiware.ManifestSource{
			Files: []string{filename},
		}
	}

	return checkTargets(targetNames, output, outfile, manifest)
}

func checkTargets(targets map[string]bool, output outputFile, writer serialize.Writer, manifest serialize.Serializable) error {
	// Identify how many verticals we missed
	misses := make([]string, 0, len(targets))
	for target, match := range targets {
		if !match {
			misses = append(misses, target)
		}
	}

	// If we got at least one vertical, save the output
	if len(targets) > len(misses) {
		if err := output.Encode(writer, manifest, nil); err != nil {
			return err
		}
	}

	// If we failed at least one vertical, return an error
	if len(misses) > 0 {
		return fmt.Errorf("failed to dump following targets: %s", strings.Join(misses, ", "))
	}
	return nil
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
