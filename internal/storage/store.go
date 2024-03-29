package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// Store manages storage of items
type Store struct {
	Path string
}

// New Store saving thing in a path
func New(path string) *Store {
	return &Store{Path: path}
}

var emptyList = make([]string, 0)

func (s *Store) readDir(assetPath string, isDir bool) ([]string, error) {
	rd, err := os.ReadDir(assetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return emptyList, nil
		}
		return nil, err
	}
	assets := make([]string, 0, len(rd))
	for _, entry := range rd {
		if entry.IsDir() == isDir {
			assets = append(assets, entry.Name())
		}
	}
	return assets, nil
}

// Assets in a particular context and resourceType
func (s *Store) Assets(context, resourceType string) ([]string, error) {
	assets, err := s.readDir(filepath.Join(s.Path, context, resourceType), true)
	if err != nil {
		return nil, err
	}
	slices.Sort(assets)
	return assets, nil
}

// Snapshots of a particular asset
func (s *Store) Snapshots(context, resourceType, asset string) ([]string, error) {
	snapshots, err := s.readDir(filepath.Join(s.Path, context, resourceType, asset), false)
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(snapshots)))
	return snapshots, nil
}

// Load item in a particular snapshot
func (s *Store) Load(context, resourceType, asset, snapshot string) ([]byte, error) {
	snap := filepath.Join(s.Path, context, resourceType, asset, snapshot)
	data, err := os.Open(snap)
	if err != nil {
		return nil, err
	}
	defer data.Close()
	return io.ReadAll(data)
}

// Save a snapshot
func (s *Store) SaveSnapshot(context, resourceType, asset, snapshot string, r io.Reader) error {
	assetPath := filepath.Join(s.Path, context, resourceType, asset)
	if err := os.MkdirAll(assetPath, 0755); err != nil {
		return err
	}
	snapFile := filepath.Join(assetPath, snapshot)
	safe := fmt.Sprintf("%s.%d", snapFile, time.Now().UnixNano())
	outfile, err := os.Create(safe)
	if err != nil {
		return err
	}
	if _, err := io.Copy(outfile, r); err != nil {
		outfile.Close()
		os.Remove(safe)
		return err
	}
	if err := outfile.Close(); err != nil {
		os.Remove(safe)
		return err
	}
	if err := os.Rename(safe, snapFile); err != nil {
		os.Remove(safe)
		return err
	}
	return nil
}

// Save creating a new Snapshot
func (s *Store) Save(context, resourceType, asset string, r io.Reader) (string, error) {
	// custom format for dates, windows does not like the
	// ":", " +XX:XX" characters in hours or timezones.
	snapshot_name := fmt.Sprintf("%s.json", time.Now().Format("20060102-150405"))
	return snapshot_name, s.SaveSnapshot(context, resourceType, asset, snapshot_name, r)
}

// Remove a snapshot
func (s *Store) RemoveSnapshot(context, resourceType, asset, snapshot string) error {
	assetFolder := filepath.Join(s.Path, context, resourceType, asset)
	snapFile := filepath.Join(assetFolder, snapshot)
	if err := os.Remove(snapFile); err != nil {
		return err
	}
	files, err := os.ReadDir(assetFolder)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return os.Remove(assetFolder)
	}
	return nil
}

// Rename a snapshot
func (s *Store) RenameSnapshot(context, resourceType, asset, oldSnapshot, newSnapshot string) error {
	assetFolder := filepath.Join(s.Path, context, resourceType, asset)
	oldSnapFile := filepath.Join(assetFolder, oldSnapshot)
	newSnapFile := filepath.Join(assetFolder, newSnapshot)
	if err := os.Rename(oldSnapFile, newSnapFile); err != nil {
		return err
	}
	return nil
}

type patchRequest struct {
	Name string `json:"name"`
}

func (s *Store) Serve() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer func() {
				io.Copy(io.Discard, r.Body)
				r.Body.Close()
			}()
		}
		urlPath := strings.Split(strings.Trim(path.Clean(r.URL.Path), "/"), "/")
		if len(urlPath) > 4 {
			http.Error(w, "Path too long", http.StatusBadRequest)
			return
		}
		for _, item := range urlPath {
			if item == "" || strings.HasPrefix(item, ".") {
				http.Error(w, "invalid empty component in path", http.StatusBadRequest)
				return
			}
		}
		if r.Method == http.MethodPost {
			if len(urlPath) < 3 {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			var (
				snapshot string
				err      error
			)
			if len(urlPath) == 3 {
				snapshot, err = s.Save(urlPath[0], urlPath[1], urlPath[2], r.Body)
			} else {
				snapshot = urlPath[3]
				err = s.SaveSnapshot(urlPath[0], urlPath[1], urlPath[2], urlPath[3], r.Body)
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(snapshot))
			return
		}
		if r.Method == http.MethodGet {
			if len(urlPath) < 2 {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			if len(urlPath) == 4 {
				data, err := s.Load(urlPath[0], urlPath[1], urlPath[2], urlPath[3])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(data)
				return
			}
			var (
				listing []string
				err     error
			)
			if len(urlPath) == 3 {
				listing, err = s.Snapshots(urlPath[0], urlPath[1], urlPath[2])
			} else {
				listing, err = s.Assets(urlPath[0], urlPath[1])
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(listing)
			return
		}
		if r.Method == http.MethodDelete {
			if len(urlPath) < 4 {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			err := s.RemoveSnapshot(urlPath[0], urlPath[1], urlPath[2], urlPath[3])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPatch {
			if len(urlPath) < 4 {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			decoder := json.NewDecoder(r.Body)
			var req patchRequest
			if err := decoder.Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			safe := filepath.Base(filepath.Clean(req.Name))
			if safe == "" || safe == "." || safe != req.Name {
				err := fmt.Errorf("'%s' is not a safe file name. Remove dots, slashes and any other unsafe character", req.Name)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			err := s.RenameSnapshot(urlPath[0], urlPath[1], urlPath[2], urlPath[3], req.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}
