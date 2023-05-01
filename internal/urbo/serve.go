package urbo

import (
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
	http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	return
}
