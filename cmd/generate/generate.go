package main

import (
	"bufio"
	"bytes"
	"go/format"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/warpcomdev/fiware"
)

type isEmpty interface {
	IsEmpty() bool
}

type generator struct {
	visited map[string]struct{}
}

// Escribe la función "serialize" de una estructura
func (g *generator) serialize(t reflect.Type, w io.StringWriter) {
	type pendingData struct {
		Name string
		Type reflect.Type
	}
	// Use ordered list instead of map so that types as generated
	// always in the same order they are met.
	pending := make([]pendingData, 0, 16)
	w.WriteString("func (x " + t.Name() + ") MarshalJSON() ([]byte, error) {\n")
	w.WriteString("  return serialize.MarshalJSON(x)\n")
	w.WriteString("}\n\n")
	w.WriteString("func (x " + t.Name() + ") Serialize(s serialize.Serializer) {\n")
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Name
		_typ := f.Type
		tags := f.Tag
		// Reuse json tags for field names
		jsonTags := strings.Split(tags.Get("json"), ",")
		if jsonTags[0] == "-" {
			continue
		}
		jsonName := jsonTags[0]
		omitempty := false
		if len(jsonTags) > 1 && jsonTags[1] == "omitempty" {
			omitempty = true
		}
		compact := tags.Get("compact")
		if compact != "true" {
			compact = "false"
		}
		switch {
		case _typ.Kind() == reflect.String:
			if omitempty {
				w.WriteString("if x." + name + " != \"\" {\n")
			}
			w.WriteString("s.KeyString(\"" + jsonName + "\", x." + name + ")\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Int:
			if omitempty {
				w.WriteString("if x." + name + " != 0 {\n")
			}
			w.WriteString("s.KeyInt(\"" + jsonName + "\", x." + name + ")\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Float64:
			if omitempty {
				w.WriteString("if x." + name + " != 0 {\n")
			}
			w.WriteString("s.KeyFloat(\"" + jsonName + "\", x." + name + ")\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Bool:
			if omitempty {
				w.WriteString("if x." + name + " {\n")
			}
			w.WriteString("s.KeyBool(\"" + jsonName + "\", x." + name + ")\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Slice && _typ.Elem().Kind() == reflect.Uint8: // json.RawMessage
			if omitempty {
				w.WriteString("if len(x." + name + ") > 0 {\n")
			}
			w.WriteString("s.KeyRaw(\"" + jsonName + "\", x." + name + ", " + compact + ")\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Slice: // other slices
			if omitempty {
				w.WriteString("if len(x." + name + ") > 0 {\n")
			}
			w.WriteString("s.BeginList(\"" + jsonName + "\")\n")
			w.WriteString("for _, y := range x." + name + " {\n")
			innerKind := _typ.Elem().Kind()
			switch {
			case innerKind == reflect.String:
				w.WriteString("s.String(y)\n")
			case innerKind == reflect.Struct:
				pending = append(pending, pendingData{Name: _typ.Elem().Name(), Type: _typ.Elem()})
				w.WriteString("s.BeginBlock(\"\"); s.Serialize(y); s.EndBlock();\n")
			default:
				log.Fatalf("Unknown slice type: %s", innerKind)
			}
			w.WriteString("}\n")
			w.WriteString("s.EndList()\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Map: // other maps
			if omitempty {
				w.WriteString("if len(x." + name + ") > 0 {\n")
			}
			w.WriteString("s.BeginBlock(\"" + jsonName + "\")\n")
			w.WriteString("for k, v := range x." + name + " {\n")
			innerKind := _typ.Elem().Kind()
			switch {
			case innerKind == reflect.String:
				w.WriteString("s.KeyString(k, v)\n")
			case innerKind == reflect.Slice: // json.RawMessage
				w.WriteString("s.KeyRaw(k, v, " + compact + ")\n")
			case innerKind == reflect.Struct:
				pending = append(pending, pendingData{Name: _typ.Elem().Name(), Type: _typ.Elem()})
				w.WriteString("s.BeginBlock(k); s.Serialize(v); s.EndBlock();\n")
			default:
				log.Fatalf("unknown map type: %s", innerKind)
			}
			w.WriteString("}\n")
			w.WriteString("s.EndBlock()\n")
			if omitempty {
				w.WriteString("}\n")
			}
		case _typ.Kind() == reflect.Struct:
			pending = append(pending, pendingData{Name: _typ.Name(), Type: _typ})
			conditional := false
			if _typ.Implements(reflect.TypeOf((*isEmpty)(nil)).Elem()) {
				w.WriteString("if !x." + name + ".IsEmpty() {\n")
				conditional = true
			}
			if !f.Anonymous {
				w.WriteString("s.BeginBlock(\"" + jsonName + "\")\n")
			}
			w.WriteString("x." + name + ".Serialize(s)\n")
			if !f.Anonymous {
				w.WriteString("s.EndBlock()\n")
			}
			if conditional {
				w.WriteString("}\n")
			}
		}
	}
	w.WriteString("}\n")
	for _, visiting := range pending {
		if _, ok := g.visited[visiting.Name]; !ok {
			g.visited[visiting.Name] = struct{}{}
			w.WriteString("\n")
			g.serialize(visiting.Type, w)
		}
	}
}

func main() {

	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	g := &generator{
		visited: make(map[string]struct{}),
	}
	w.WriteString("package fiware\n\n")
	w.WriteString("import (\n\"github.com/warpcomdev/fiware/internal/serialize\"\n)\n\n")
	w.WriteString("// Autogenerated file - DO NOT EDIT\n\n")
	g.serialize(reflect.TypeOf(fiware.Vertical{}), w)
	w.Flush()

	rawCode, filename, result := buffer.Bytes(), "serializations.go", 0
	fmtCode, err := format.Source(rawCode)
	if err != nil {
		log.Printf("Failed to format code code: %v\n", err)
		fmtCode, filename, result = rawCode, "serializations.go.fail", -1
	}

	outfile, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to open generated file: %v", err)
	}
	defer outfile.Close()
	outfile.Write(fmtCode)
	if result != 0 {
		os.Exit(result)
	}
}
