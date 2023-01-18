package config

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func (s *Store) Server() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer exhaust(r)
		id := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
		if r.Method == http.MethodGet {
			if id == "" {
				s.onList(w, r)
				return
			}
			s.onLoad(w, r, id)
			return
		}
		if r.Method == http.MethodPost {
			if id != "" {
				http.Error(w, "do not send context id for POST", http.StatusMethodNotAllowed)
				return
			}
			s.onSave(w, r)
			return
		}
		if r.Method == http.MethodDelete {
			if id == "" {
				http.Error(w, "missing context id", http.StatusMethodNotAllowed)
			}
			s.onRemove(w, r, id)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func exhaust(r *http.Request) {
	if r.Body != nil {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
}

func reply(w http.ResponseWriter, data interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.Encode(data)
}

func (s *Store) onList(w http.ResponseWriter, r *http.Request) {
	listing, err := s.List(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reply(w, listing)
}

func (s *Store) onLoad(w http.ResponseWriter, r *http.Request, id string) {
	info, err := s.Info(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Clients must authenticate, they cannot reuse common tokens
	info.Token = ""
	info.UrboToken = ""
	reply(w, info)
}

func (s *Store) onSave(w http.ResponseWriter, r *http.Request) {
	cfg, err := FromBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg.Token = ""
	cfg.UrboToken = ""
	if err := s.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reply(w, cfg)
}

func (s *Store) onRemove(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reply(w, id)
}

func FromHeaders(r *http.Request, store *Store) (Config, error) {
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
			return Config{}, errors.New("Missing header Fiware-Context or username")
		}
		if token == "" {
			return Config{}, errors.New("Missing header X-Auth-Token or password")
		}
	}
	selected, err := store.Info(context)
	if err != nil {
		return Config{}, err
	}
	selected.Token = token
	selected.UrboToken = token
	return selected, nil
}

func FromBody(r *http.Request) (Config, error) {
	var cfg Config
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return cfg, errors.New("Invalid content-type")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 65536))
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
