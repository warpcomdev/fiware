package template

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/go-jsonnet"
	"go.starlark.net/starlark"
)

//go:embed builtin/*.tmpl
var builtin embed.FS

func newTemplate() (*template.Template, error) {
	// Add 'include' function to be able to indent templates
	makeFuncMap := func(t *template.Template) template.FuncMap {
		funcMap := make(template.FuncMap)
		// copied from: https://github.com/helm/helm/blob/8648ccf5d35d682dcd5f7a9c2082f0aaf071e817/pkg/engine/engine.go#L147-L154
		funcMap["include"] = func(name string, data interface{}) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := t.ExecuteTemplate(buf, name, data); err != nil {
				return "", err
			}
			return buf.String(), nil
		}
		return funcMap
	}

	// First, add built-in templates
	var (
		tpl = template.New("builtin").Funcs(sprig.TxtFuncMap())
		err error
	)
	if tpl, err = tpl.Funcs(makeFuncMap(tpl)).ParseFS(builtin, "builtin/*.tmpl"); err != nil {
		return nil, fmt.Errorf("failed to load built-in templates: %w", err)
	}
	return tpl, nil
}

func Load(datafile string, params map[string]string, output interface{}) error {
	if datafile != "" {
		// Use starlark for .star or .py files
		lowerName := strings.ToLower(datafile)
		if strings.HasSuffix(lowerName, ".star") || strings.HasSuffix(lowerName, ".py") {
			return loadStarlark(datafile, params, output)
		}
		vm := jsonnet.MakeVM()
		for k, v := range params {
			vm.ExtVar(k, v)
		}
		jsonStr, err := vm.EvaluateFile(datafile)

		if err != nil {
			return fmt.Errorf("failed to load file %s as jsonnet: %w", datafile, err)
		}
		if err := json.Unmarshal([]byte(jsonStr), output); err != nil {
			return fmt.Errorf("failed to unmarshal file %s as jsonnet: %w", datafile, err)
		}
	}
	return nil
}

func loadStarlark(datafile string, params map[string]string, output interface{}) error {
	// Execute Starlark program in a file.
	thread := &starlark.Thread{Name: "datafile"}
	globals, err := starlark.ExecFile(thread, datafile, nil, nil)
	if err != nil {
		return err
	}
	base, ext := path.Base(datafile), path.Ext(datafile)
	base = base[0 : len(base)-len(ext)]
	data, ok := globals[base]
	if !ok {
		return fmt.Errorf("datafile %s should have a global variable %s", datafile, base)
	}
	if _, ok := data.(starlark.Callable); ok {
		ext := starlark.NewDict(len(params))
		for k, v := range params {
			ext.SetKey(starlark.String(k), starlark.String(v))
		}
		result, err := starlark.Call(thread, data, starlark.Tuple{ext}, nil)
		if err != nil {
			return err
		}
		data = result
	}
	valBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(valBytes, &output)
}

type stringError string

func (s stringError) Error() string {
	return string(s)
}

const (
	ErrFailedParamsNoMap    stringError = "failed to insert params, data is not a map"
	ErrFailedParamsExisting stringError = "failed to insert params, `params` key already exists"
)

func Render(datafile string, templates []string, params map[string]string, output io.Writer) error {

	// First, add built-in templates
	tpl, err := newTemplate()
	if err != nil {
		return err
	}

	var data interface{}
	if err := Load(datafile, params, &data); err != nil {
		return err
	}
	if params != nil {
		m, ok := data.(map[string]interface{})
		if !ok {
			return ErrFailedParamsNoMap
		}
		if _, ok := m["params"]; ok {
			return ErrFailedParamsExisting
		}
		m["params"] = params
		data = m
	}

	// Then, any other file
	other_files := make([]string, 0, len(templates))
	for _, arg := range templates {
		if prev := tpl.Lookup(arg); prev == nil {
			other_files = append(other_files, arg)
		}
	}
	if len(other_files) > 0 {
		tpl, err = tpl.ParseFiles(other_files...)
		if err != nil {
			return fmt.Errorf("failed to load templates %s: %w", other_files, err)
		}
	}

	// We only run the first template in the list
	selected := path.Base(templates[0])
	if prev := tpl.Lookup(selected); prev == nil {
		return fmt.Errorf("template %s not found. %s", selected, tpl.DefinedTemplates())
	}

	if err := tpl.ExecuteTemplate(output, selected, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templates[0], err)
	}
	return nil
}

// Builtins returns the list of builtin templates
func Builtins() ([]string, error) {
	tpl, err := newTemplate()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tpl.Templates()))
	for _, tpl := range tpl.Templates() {
		names = append(names, tpl.Name())
	}
	return names, nil
}
