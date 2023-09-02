package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/snapshots"
	"github.com/warpcomdev/fiware/internal/storage"
	"github.com/warpcomdev/fiware/internal/urbo"
)

//go:embed static/*
var staticFS embed.FS

// prepare Server and address to start http rest api
func prepareServer(currentStore *config.Store, c *cli.Context, backoff keystone.ExponentialBackoff) (http.Handler, string, error) {

	client := httpClient(0, 15*time.Second)
	mux := &http.ServeMux{}
	configDir, err := currentStore.GetConfigDir()
	if err != nil {
		return nil, "", err
	}
	storageDir := filepath.Join(configDir, "storage")
	mux.Handle("/api/auth", cors(authServe(client, currentStore, backoff)))
	mux.Handle("/api/contexts/", cors(http.StripPrefix("/api/contexts", currentStore.Server())))
	mux.Handle("/api/snaps/", cors(http.StripPrefix("/api/snaps", snapshots.Serve(client, currentStore))))
	mux.Handle("/api/urbo/", cors(http.StripPrefix("/api/urbo", urbo.Serve(client, currentStore))))
	mux.Handle("/api/storage/", cors(http.StripPrefix("/api/storage", storage.New(storageDir).Serve())))
	mux.Handle("/legacy", legacyHandler())
	var serveFS fs.FS
	if c.NArg() > 0 {
		subdir := c.Args().First()
		serveFS = os.DirFS(subdir)
	} else {
		serveFS, err = fs.Sub(staticFS, "static")
		if err != nil {
			panic(err)
		}
	}
	mux.Handle("/", http.FileServer(http.FS(serveFS)))
	port := c.Int(portFlag.Name)
	var addr string
	if port <= 0 {
		// If application received no command, port is 0.
		// listen on localhost to avoid problems with
		// windows firewall
		port = 9181
		addr = fmt.Sprintf("localhost:%d", port)
	} else {
		addr = fmt.Sprintf(":%d", port)
	}
	fmt.Printf("Listening at addr %s\n", addr)
	return mux, addr, nil
}
