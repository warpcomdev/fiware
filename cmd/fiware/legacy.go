package main

import (
	"archive/zip"
	"bufio"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/warpcomdev/fiware/internal/decode"
)

//go:embed legacy/*
var legacy embed.FS

func legacyHandler() http.Handler {
	sub, err := fs.Sub(legacy, "legacy")
	if err != nil {
		panic("Failed to read legacy files")
	}
	fsHandler := http.FileServer(http.FS(sub))
	onRenderRequest := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fsHandler.ServeHTTP(w, r)
		} else if r.Method == "POST" {
			onPostRenderRequest(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
	return http.HandlerFunc(onRenderRequest)
}

func onPostRenderRequest(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r.Body == nil {
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
	}()
	// Read form fields
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %s", err.Error()), http.StatusBadRequest)
		return
	}
	vertical := r.FormValue("vertical")
	log.Print("Vertical: ", vertical)
	subservice := r.FormValue("subservice")
	log.Print("Subservice: ", subservice)
	data := r.FormValue("data")
	log.Print("Data: ", data)
	// fund out if input is NGSI or CSV
	scanner := bufio.NewScanner(strings.NewReader(data))
	scanner.Split(bufio.ScanLines)
	if !scanner.Scan() {
		http.Error(w, "No data received", http.StatusBadRequest)
		return
	}
	line := strings.TrimSpace(scanner.Text())
	var ext string
	if strings.HasPrefix(line, "entityID") {
		ext = "csv"
	} else {
		ext = "md"
	}
	// Create temporary directory and file
	tmpDir, err := os.MkdirTemp("", "fiware")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp folder: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)
	tmpFile, err := os.Create(filepath.Join(tmpDir, "data."+ext))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp file: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()
	for {
		if _, err := tmpFile.WriteString(line + "\n"); err != nil {
			http.Error(w, fmt.Sprintf("failed to write to temp file: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if !scanner.Scan() {
			break
		}
		line = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	if err := tmpFile.Close(); err != nil {
		http.Error(w, fmt.Sprintf("failed to close temp file: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	outFile := filepath.Join(tmpDir, "out.cue")
	if err := decode.Decode(outFile, vertical, subservice, []string{tmpFile.Name()}, ""); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode input data: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	outDir := filepath.Join(tmpDir, "out")
	if err := os.Mkdir(outDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to create output model folder: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	err = renderTemplate(outFile, "", outDir, "json", []string{"default_model.tmpl"}, map[string]string{})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to render template: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=models.zip")
	w.WriteHeader(http.StatusOK)
	zw := zip.NewWriter(w)
	defer zw.Close()
	if err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		fileName := filepath.Base(path)
		f, err := zw.Create(fileName)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, input); err != nil {
			return err
		}
		return nil
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to compress folder: %s", err.Error()), http.StatusInternalServerError)
	}
}
