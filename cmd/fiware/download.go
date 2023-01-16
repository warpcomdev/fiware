package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/serialize"
	"github.com/warpcomdev/fiware/internal/snapshots"
	"github.com/warpcomdev/fiware/internal/urbo"
)

type verticalDownloader struct {
	Selected  config.Config
	Manifest  fiware.Manifest
	Client    keystone.HTTPClient
	Urbo      *urbo.Urbo
	Headers   http.Header
	Verticals map[string]string
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
	downloader.Verticals = make(map[string]string)
	for _, item := range downloader.Manifest.Verticals {
		downloader.Verticals[item.Slug] = item.Name
	}
	return &downloader, nil
}

// List all available verticals as strings "name (slug)"
func (v *verticalDownloader) List() ([]string, error) {
	names := make([]string, 0, len(v.Verticals))
	for slug, name := range v.Verticals {
		names = append(names, fmt.Sprintf("%s (%s)", name, slug))
	}
	sort.Sort(sort.StringSlice(names))
	return names, nil
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
	manifestPrefixed := "urbo-" + v.Slug
	manifestFilename := manifestPrefixed + ".json"
	manifestFullname := outputFile(filepath.Join(outdir, manifestFilename))
	manifestFile, err := manifestFullname.Create()
	if err != nil {
		return "", err
	}
	defer manifestFile.Close()

	paneldir := filepath.Join(outdir, manifestPrefixed)
	panelCount := len(v.Panels) + len(v.ShadowPanels)
	sources := fiware.ManifestSource{
		Path:  v.Slug,
		Files: make([]string, 0, panelCount),
	}
	if panelCount <= 0 {
		return "", nil
	}
	if err := ensureDir(paneldir); err != nil {
		return "", err
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
	manifestFullname.Encode(manifestFile, manifest, nil)
	return manifestFilename, nil
}

func listVerticals(c *cli.Context, store *config.Store) ([]string, error) {
	downloader, err := newVerticalDownloader(c, store)
	if err != nil {
		return nil, err
	}
	return downloader.List()
}

func newProjectDownloader(c *cli.Context, store *config.Store) (*snapshots.Project, error) {
	selected, err := getConfig(c, store)
	if err != nil {
		return nil, err
	}
	client := httpClient(c.Bool(verboseFlag.Name))
	keystone, headers, err := getKeystoneHeaders(c, selected)
	if err != nil {
		return nil, err
	}
	return snapshots.NewProject(selected, client, keystone, headers)
}

func listProjects(c *cli.Context, store *config.Store, manifest *fiware.Manifest) ([]string, error) {
	downloader, err := newProjectDownloader(c, store)
	if err != nil {
		return nil, err
	}
	return downloader.Names(), nil
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

func downloadVertical(c *cli.Context, store *config.Store) error {
	downloader, err := newVerticalDownloader(c, store)
	if err != nil {
		return err
	}

	allTargets := c.Bool(allFlag.Name)
	if c.NArg() <= 0 && !allTargets {
		names, err := downloader.List()
		if err != nil {
			return err
		}
		if len(names) <= 0 {
			return errors.New("failed to discover any vertical")
		}
		return fmt.Errorf("select from:\n- %s", strings.Join(names, "\n- "))
	}

	// build a mapping from name to slug(s)
	nameToSlug := make(map[string][]string)
	for slug, name := range downloader.Verticals {
		slugs, ok := nameToSlug[name]
		if !ok {
			slugs = make([]string, 0, 1)
		}
		slugs = append(slugs, slug)
		nameToSlug[name] = slugs
	}

	// Match all arguments against names or slugs
	targetSlugs := make(map[string]bool)
	for _, target := range c.Args().Slice() {
		var matchingSlugs []string
		if _, found := downloader.Verticals[target]; found {
			matchingSlugs = []string{target}
		} else {
			if slugs, found := nameToSlug[target]; found {
				matchingSlugs = slugs
			}
		}
		if len(matchingSlugs) <= 0 {
			names, err := downloader.List()
			if err != nil {
				return err
			}
			return fmt.Errorf("No matching slug or vertical for '%s'. Select from: %s", target, strings.Join(names, "\n"))
		}
		for _, slug := range matchingSlugs {
			targetSlugs[slug] = false
		}
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

	output := outputFile(filepath.Join(outdir, "00_urbo.json"))
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
		if filename != "" {
			manifest.Deployment.Sources["vertical:"+v.Slug] = fiware.ManifestSource{
				Files: []string{filename},
			}
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
		return fmt.Errorf("select a resource from: %s", strings.Join(downloader.Names(), ", "))
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

	output := outputFile(filepath.Join(outdir, "00_orion.json"))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()

	client := httpClient(c.Bool(verboseFlag.Name))
	maximum := c.Int(maxFlag.Name)
	for _, v := range downloader.Projects {
		if _, ok := targetNames[v.Name]; !ok && !allTargets {
			continue
		}
		targetNames[v.Name] = true
		// Output is saved in manifest format
		currentOutDir := outdir + v.Name
		current, err := downloader.Snap(client, v, maximum)
		if err != nil {
			return err
		}
		currentSource, err := snapshots.WriteManifest(current, currentOutDir)
		if err != nil {
			return err
		}
		currentSource.Path = "." + v.Name
		manifest.Deployment.Sources["subservice:"+v.Name] = currentSource
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
