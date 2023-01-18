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
type dldFunc func(keystone.HTTPClient, http.ResponseWriter, *http.Request, config.Config, string)

func Serve(client keystone.HTTPClient, store *config.Store) http.Handler {
	mux := &http.ServeMux{}
	mux.Handle("/projects/", http.StripPrefix("/projects", serve(client, store, projectLister, nil)))
	mux.Handle("/verticals/", http.StripPrefix("/verticals", serve(client, store, urboLister, nil)))
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
		context := r.Header.Get("Fiware-Context")
		token := r.Header.Get("X-Auth-Token")
		if context == "" || token == "" {
			username, password, ok := r.BasicAuth()
			if ok {
				if context == "" {
					context = username
				}
				if token == "" {
					token = password
				}
			}
			if context == "" {
				http.Error(w, "Missing header Fiware-Context or username", http.StatusBadRequest)
				return
			}
			if token == "" {
				http.Error(w, "Missing header X-Auth-Token or password", http.StatusUnauthorized)
				return
			}
		}
		selected, err := store.Info(context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		selected.Token = token
		selected.UrboToken = token
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
