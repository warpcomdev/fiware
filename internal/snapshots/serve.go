package snapshots

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
	"github.com/warpcomdev/fiware/internal/urbo"
)

type listFunc func(keystone.HTTPClient, config.Config) (interface{}, error)
type dldFunc func(keystone.HTTPClient, http.ResponseWriter, *http.Request, config.Config)

func Serve(client keystone.HTTPClient) http.Handler {
	mux := &http.ServeMux{}
	mux.Handle("/projects/", http.StripPrefix("/projects", serve(client, projectLister, nil)))
	mux.Handle("/verticals/", http.StripPrefix("/verticals", serve(client, urboLister, nil)))
	return mux
}

func serve(client keystone.HTTPClient, lister listFunc, dlder dldFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer exhaust(r)
		id := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}
		reader := io.LimitReader(r.Body, 65536)
		decode := json.NewDecoder(reader)
		var selected config.Config
		if err := decode.Decode(&selected); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		if dlder != nil {
			http.Error(w, "operation not supported", http.StatusNotAcceptable)
			return
		}
		dlder(client, w, r, selected)
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
