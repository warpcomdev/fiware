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
	query := r.URL.Query()
	if query.Get("context") == "" {
		http.Error(w, "must provide context name", http.StatusBadRequest)
		return
	}
	if query.Get("slug") == "" {
		http.Error(w, "must provide slug name", http.StatusBadRequest)
		return
	}
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
	context := query.Get("context")
	slug := query.Get("slug")
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
