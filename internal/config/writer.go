package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

// Generic interface for writing files
type Writer interface {
	AtomicSave(path, tmpPrefix string, data []byte) error
}

// save some file atomically
func AtomicSave(writer Writer, path string, tmpPrefix string, data interface{}) error {
	byteData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return writer.AtomicSave(path, tmpPrefix, byteData)
}

// Writer implementation for filesystem
type FolderWriter string

// save some file atomically
func (rootFolder FolderWriter) AtomicSave(path string, tmpPrefix string, byteData []byte) error {
	if string(rootFolder) != "" {
		path = filepath.Join(string(rootFolder), path)
	}
	fullFolder, _ := filepath.Split(path)
	if err := os.MkdirAll(fullFolder, 0750); err != nil {
		// Ignore error if the folder exists
		if !os.IsExist(err) {
			return err
		}
	}
	tmpFile, err := ioutil.TempFile(fullFolder, tmpPrefix)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) // just in case
	_, err = tmpFile.Write(byteData)
	closeErr := tmpFile.Close() // Close before returning and renaming, for windows.
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if runtime.GOOS == "windows" {
		os.Chmod(path, 0644) // So that os.Rename works. See https://github.com/golang/go/issues/38287
	}
	return os.Rename(tmpFile.Name(), path)
}

// PrefixWriter is a writer that appends a path prefix to every write
type WriterFunc func(path, tmpPrefix string, data []byte) error

// AtomicSave implements Writer
func (writer WriterFunc) AtomicSave(path string, tmpPrefix string, byteData []byte) error {
	return writer(path, tmpPrefix, byteData)
}

// PrefixWriter returns a writer that adds a prefix to every write
func PrefixWriter(writer Writer, prefix string) Writer {
	return WriterFunc(func(path, tmpPrefix string, data []byte) error {
		path = filepath.Join(prefix, path)
		return writer.AtomicSave(path, tmpPrefix, data)
	})
}
