package importer

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.starlark.net/starlark"
)

// loadStarlark reads a Jsonnet file with the provided params as arguments
func loadStarlark(datafile string, params map[string]string, pathLib string) (string, error) {
	// Execute Starlark program in a file.
	importer := &builtinStarlarkImporter{path: pathLib}
	thread := &starlark.Thread{
		Name: "datafile",
		Load: importer.Load,
	}
	globals, err := starlark.ExecFile(thread, datafile, nil, nil)
	if err != nil {
		return "", err
	}
	base, ext := path.Base(datafile), path.Ext(datafile)
	base = base[0 : len(base)-len(ext)]
	data, ok := globals[base]
	if !ok {
		return "", fmt.Errorf("datafile %s should have a global variable %s", datafile, base)
	}
	if _, ok := data.(starlark.Callable); ok {
		ext := starlark.NewDict(len(params))
		for k, v := range params {
			ext.SetKey(starlark.String(k), starlark.String(v))
		}
		result, err := starlark.Call(thread, data, starlark.Tuple{ext}, nil)
		if err != nil {
			return "", err
		}
		data = result
	}
	return data.String(), nil
}

type entry struct {
	globals starlark.StringDict
	err     error
}

type builtinStarlarkImporter struct {
	path  string
	cache map[string]*entry
}

// copied from https://github.com/google/starlark-go/blob/c8e9b32ba2fb0cd3f78dd181e71b013b093648ef/starlark/example_test.go
func (b *builtinStarlarkImporter) Load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	if b.cache == nil {
		b.cache = make(map[string]*entry)
	}
	tryPaths := []string{""}
	if b.path != "" {
		tryPaths = append(tryPaths, b.path)
	}
	for _, tryPath := range tryPaths {
		absPath, err := filepath.Abs(filepath.Join(tryPath, module))
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			continue // try next path
		}
		e, ok := b.cache[absPath]
		if e == nil {
			if ok {
				// request for package whose loading is in progress
				return nil, fmt.Errorf("cycle in load graph")
			}

			// Add a placeholder to indicate "load in progress".
			b.cache[module] = nil

			// Load and initialize the module in a new thread.
			thread := &starlark.Thread{Name: "exec " + module, Load: b.Load}
			globals, err := starlark.ExecFile(thread, absPath, nil, nil)
			e = &entry{globals, err}

			// Update the cache.
			b.cache[absPath] = e
		}
		return e.globals, e.err
	}
	return nil, os.ErrNotExist
}
