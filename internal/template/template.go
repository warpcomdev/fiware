package template

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/warpcomdev/fiware"
)

//go:embed builtin/*.tmpl
var builtin embed.FS

type verticalWithParams struct {
	fiware.Manifest
	Params map[string]string `json:"params,omitempty"`
}

// Turn a manifest into a json dict to use in a template
func ManifestForTemplate(manifest fiware.Manifest, params map[string]string) (interface{}, error) {
	var (
		data       interface{}
		strictData verticalWithParams
	)
	strictData.Manifest = manifest
	if len(params) > 0 {
		strictData.Params = params
	}
	// Convierto a map[string]interface{} pasando por json,
	// porque no quiero que los diseñadores de los templates
	// necesiten conocer el formato de los objetos golang.
	// Mejor que puedan trabajar con la misma estructura de atributos
	// que en el fichero de datos.
	text, err := json.Marshal(strictData)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(text, &data); err != nil {
		return nil, err
	}
	return data, nil
}

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

func Render(templates []string, data interface{}, output io.Writer) error {

	// First, add built-in templates
	tpl, err := newTemplate()
	if err != nil {
		return err
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
