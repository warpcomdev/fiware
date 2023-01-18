package snapshots

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
)

type listFunc func(keystone.HTTPClient, config.Config) (interface{}, error)
type dldFunc func(keystone.HTTPClient, http.ResponseWriter, *http.Request, config.Config, string)

func Serve(client keystone.HTTPClient, store *config.Store) http.Handler {
	mux := &http.ServeMux{}
	mux.Handle("/projects/", http.StripPrefix("/projects", serve(client, store, projectLister, projectDownloader)))
	mux.Handle("/verticals/", http.StripPrefix("/verticals", serve(client, store, urboLister, urboDownloader)))
	return mux
}

func serve(client keystone.HTTPClient, store *config.Store, lister listFunc, dlder dldFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer exhaust(r)
		id := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		selected, err := config.FromHeaders(r, store)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if id == "" {
			if lister == nil {
				http.Error(w, "operation not supported", http.StatusNotAcceptable)
				return
			}
			data, err := lister(client, selected)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			enc := json.NewEncoder(w)
			enc.Encode(data)
			return
		}
		if dlder == nil {
			http.Error(w, "operation not supported", http.StatusNotAcceptable)
			return
		}
		dlder(client, w, r, selected, id)
	})
}

func exhaust(r *http.Request) {
	if r.Body != nil {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
}

func projectLister(client keystone.HTTPClient, selected config.Config) (interface{}, error) {
	api, err := keystone.New(selected.KeystoneURL, selected.Username, selected.Service)
	if err != nil {
		return nil, err
	}
	headers := api.Headers(selected.Subservice, selected.Token)
	return api.Projects(client, headers)
}

func urboLister(client keystone.HTTPClient, selected config.Config) (interface{}, error) {
	api, err := urbo.New(selected.UrboURL, selected.Username, selected.Service, selected.Service)
	if err != nil {
		return nil, err
	}
	headers := api.Headers(selected.UrboToken)
	return api.GetVerticals(client, headers)
}

func projectDownloader(client keystone.HTTPClient, w http.ResponseWriter, r *http.Request, selected config.Config, id string) {
	api, err := keystone.New(selected.KeystoneURL, selected.Username, selected.Service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(id, "/") {
		id = "/" + id
	}
	headers := api.Headers(id, selected.Token)
	manifest, err := Project(client, api, selected, headers, fiware.Project{Name: id}, 10000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	attachName := strings.TrimPrefix(id, "/")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", attachName))
	w.Header().Set("Content-Type", "application/zip")
	w.WriteHeader(http.StatusOK)
	zipper := &zipWriter{
		Visited: make(map[string]struct{}),
		Zipper:  zip.NewWriter(w),
	}
	defer zipper.Zipper.Close()
	source, err := WriteManifest(manifest, nil, zipper)
	if err != nil {
		log.Print(err.Error())
		return
	}
	deployment := fiware.Manifest{
		Deployment: fiware.DeploymentManifest{
			Sources: map[string]fiware.ManifestSource{
				attachName: source,
			},
		},
	}
	if err := config.AtomicSave(zipper, "project.json", "project", deployment); err != nil {
		log.Print(err.Error())
	}
}

func urboDownloader(client keystone.HTTPClient, w http.ResponseWriter, r *http.Request, selected config.Config, id string) {
	api, err := urbo.New(selected.UrboURL, selected.Username, selected.Service, selected.Service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id = strings.TrimPrefix(id, "/")
	headers := api.Headers(selected.UrboToken)
	manifest, panels, err := Urbo(client, api, selected, headers, fiware.Vertical{Slug: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	attachName := id
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", attachName))
	w.Header().Set("Content-Type", r.Header.Get("application/zip"))
	w.WriteHeader(http.StatusOK)
	zipper := &zipWriter{
		Visited: make(map[string]struct{}),
		Zipper:  zip.NewWriter(w),
	}
	defer zipper.Zipper.Close()
	if _, err := WriteManifest(manifest, panels, zipper); err != nil {
		log.Print(err.Error())
		return
	}
}

type zipWriter struct {
	Visited map[string]struct{}
	Zipper  *zip.Writer
}

func (z *zipWriter) AtomicSave(path, tmpPrefix string, data []byte) error {
	folder, _ := filepath.Split(path)
	if folder != "" {
		if _, ok := z.Visited[folder]; !ok {
			if _, err := z.Zipper.Create(folder + "/"); err != nil {
				return err
			}
			z.Visited[folder] = struct{}{}
		}
	}
	w, err := z.Zipper.Create(path)
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err != nil {
		return err
	}
	if n < len(data) {
		return errors.New("failed to zip all available data")
	}
	return nil
}
