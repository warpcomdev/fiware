package main

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
)

func uploadResource(c *cli.Context, store *config.Store) error {

	if c.NArg() <= 0 {
		return errors.New("login first and then select a resource to upload")
	}

	selected, err := getConfig(c, store)
	if err != nil {
		return err
	}

	u, header, err := getUrboHeaders(c, &selected)
	if err != nil {
		return err
	}

	client := httpClient(verbosity(c))
	for _, target := range c.Args().Slice() {
		fullpath, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		dirname, filename := filepath.Split(fullpath)
		if err := uploadPanel(u, client, header, os.DirFS(dirname), filename); err != nil {
			return err
		}
	}
	return nil
}

func uploadPanel(u *urbo.Urbo, client keystone.HTTPClient, header http.Header, fsys fs.FS, path string) error {
	file, err := fsys.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	return u.UploadPanel(client, header, bytes)
}
