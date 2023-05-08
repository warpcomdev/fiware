package urbo

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/warpcomdev/fiware/internal/config"
	"github.com/warpcomdev/fiware/internal/keystone"
)

func Serve(client keystone.HTTPClient, store *config.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		servePanels(client, store, w, r)
	})
}

func servePanels(client keystone.HTTPClient, store *config.Store, w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer func() {
			io.Copy(ioutil.Discard, r.Body)
			r.Body.Close()
		}()
	}
	path := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 2)
	if len(path) < 2 {
		http.Error(w, "path must include context name and slug", http.StatusBadRequest)
		return
	}
	context := path[0]
	slug := path[1]
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		http.Error(w, "must provide authorization header", http.StatusUnauthorized)
		return
	}
	if !strings.HasPrefix(bearer, "Bearer ") {
		http.Error(w, "invalid authorization header", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(bearer, "Bearer ")
	contextObj, err := store.Info(context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api, err := New(contextObj.UrboURL, contextObj.Username, contextObj.Service, contextObj.Service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	headers := api.Headers(token)
	if r.Method == http.MethodGet {
		msg, err := api.DownloadPanel(client, headers, slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(msg)
		return
	}
	if r.Method == http.MethodPost {
		panel, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// test it is valid json
		var dummy map[string]interface{}
		if err := json.Unmarshal(panel, &dummy); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Test the slug matches
		dummySlug, ok := dummy["slug"]
		if !ok {
			http.Error(w, "panel must have slug", http.StatusBadRequest)
			return
		}
		dummySlugString, ok := dummySlug.(string)
		if !ok {
			http.Error(w, "panel slug must be string", http.StatusBadRequest)
			return
		}
		if slug != dummySlugString {
			http.Error(w, "panel slug does not match path", http.StatusBadRequest)
			return
		}
		if err := api.UploadPanel(client, headers, panel); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	return
}
