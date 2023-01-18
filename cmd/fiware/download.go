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
	"github.com/warpcomdev/fiware/internal/serialize"
	"github.com/warpcomdev/fiware/internal/snapshots"
	"github.com/warpcomdev/fiware/internal/urbo"
)

type verticalDownloader struct {
	Selected  config.Config
	Api       *urbo.Urbo
	Headers   http.Header
	Client    keystone.HTTPClient
	Verticals map[string]fiware.Vertical
}

func newVerticalDownloader(c *cli.Context, store *config.Store) (*verticalDownloader, error) {
	selected, err := getConfig(c, store)
	if err != nil {
		return nil, err
	}
	api, headers, err := getUrboHeaders(c, &selected)
	if err != nil {
		return nil, err
	}
	client := httpClient(c.Bool(verboseFlag.Name))
	verticals, err := api.GetVerticals(client, headers)
	if err != nil {
		return nil, err
	}
	return &verticalDownloader{
		Selected:  selected,
		Api:       api,
		Headers:   headers,
		Client:    client,
		Verticals: verticals,
	}, nil
}

func (v *verticalDownloader) List() []string {
	return snapshots.VerticalList(v.Verticals)
}

func (dld *verticalDownloader) Download(c *cli.Context, store *config.Store) error {

	allTargets := c.Bool(allFlag.Name)
	if c.NArg() <= 0 && !allTargets {
		names := dld.List()
		if len(names) <= 0 {
			return errors.New("failed to discover any vertical")
		}
		return fmt.Errorf("select from:\n- %s", strings.Join(names, "\n- "))
	}

	// build a mapping from name to slug(s)
	nameToSlug := make(map[string][]string)
	for slug, vertical := range dld.Verticals {
		slugs, ok := nameToSlug[vertical.Name]
		if !ok {
			slugs = make([]string, 0, 1)
		}
		slugs = append(slugs, slug)
		nameToSlug[vertical.Name] = slugs
	}

	// Match all arguments against names or slugs
	targetSlugs := make(map[string]bool)
	for _, target := range c.Args().Slice() {
		var matchingSlugs []string
		if _, found := dld.Verticals[target]; found {
			matchingSlugs = []string{target}
		} else {
			if slugs, found := nameToSlug[target]; found {
				matchingSlugs = slugs
			}
		}
		if len(matchingSlugs) <= 0 {
			names := dld.List()
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

	output := outputFile(filepath.Join(outdir, "urbo.json"))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()

	for _, v := range dld.Verticals {
		if _, ok := targetSlugs[v.Slug]; !ok && !allTargets {
			continue
		}
		targetSlugs[v.Slug] = true
		// Output is saved in manifest format
		current, panels, err := snapshots.Urbo(dld.Client, dld.Api, dld.Selected, dld.Headers, v)
		if err != nil {
			return err
		}
		currentOutDir := filepath.Join(outdir, v.Slug)
		currentSource, err := snapshots.WriteManifest(current, panels, config.FolderWriter(currentOutDir))
		if err != nil {
			return err
		}
		currentSource.Path = "./" + v.Slug
		manifest.Deployment.Sources["urboverticals:"+v.Slug] = currentSource
	}

	return checkTargets(targetSlugs, output, outfile, manifest)
}

type projectDownloader struct {
	Selected config.Config
	Api      *keystone.Keystone
	Headers  http.Header
	Client   keystone.HTTPClient
	Projects []fiware.Project
}

func newProjectDownloader(c *cli.Context, store *config.Store) (*projectDownloader, error) {
	selected, err := getConfig(c, store)
	if err != nil {
		return nil, err
	}
	api, headers, err := getKeystoneHeaders(c, &selected)
	if err != nil {
		return nil, err
	}
	client := httpClient(c.Bool(verboseFlag.Name))
	projects, err := api.Projects(client, headers)
	if err != nil {
		return nil, err
	}
	return &projectDownloader{
		Selected: selected,
		Api:      api,
		Headers:  headers,
		Client:   client,
		Projects: projects,
	}, nil
}

func (dld *projectDownloader) List() []string {
	return snapshots.ProjectList(dld.Projects)
}

func (dld *projectDownloader) Download(c *cli.Context, store *config.Store) error {

	allTargets := c.Bool(allFlag.Name)
	if c.NArg() <= 0 && !allTargets {
		return fmt.Errorf("select a resource from: %s", strings.Join(dld.List(), ", "))
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

	output := outputFile(filepath.Join(outdir, "orion.json"))
	outfile, err := output.Create()
	if err != nil {
		return err
	}
	defer outfile.Close()

	maximum := c.Int(maxFlag.Name)
	for _, v := range dld.Projects {
		if _, ok := targetNames[v.Name]; !ok && !allTargets {
			continue
		}
		targetNames[v.Name] = true
		// Output is saved in manifest format
		currentOutDir := filepath.Join(outdir, v.Name)
		current, err := snapshots.Project(dld.Client, dld.Api, dld.Selected, dld.Headers, v, maximum)
		if err != nil {
			return err
		}
		currentSource, err := snapshots.WriteManifest(current, nil, config.FolderWriter(currentOutDir))
		if err != nil {
			return err
		}
		currentSource.Path = "." + v.Name
		manifest.Deployment.Sources["subservice:"+v.Name] = currentSource
	}

	return checkTargets(targetNames, output, outfile, manifest)
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
