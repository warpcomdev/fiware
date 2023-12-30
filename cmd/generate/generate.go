package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	cueformat "cuelang.org/go/cue/format"
	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/serialize"
)

type isEmpty interface {
	IsEmpty() bool
}

type isHook interface {
	SerializeHook(string, serialize.Serializer)
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
		sort := tags.Get("sort")
		if sort != "true" {
			sort = "false"
		}
		switch {
		case _typ.Kind() == reflect.String:
			if omitempty {
				w.WriteString("if x." + name + " != \"\" {\n")
			}
			w.WriteString("s.KeyString(\"" + jsonName + "\", string(x." + name + "))\n")
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
			innerKind := _typ.Elem().Kind()
			if innerKind == reflect.String && sort == "true" {
				// If the slice is of string, sort first
				w.WriteString("for _, y := range serialize.Sorted(x." + name + ") {\n")
			} else {
				w.WriteString("for _, y := range x." + name + " {\n")
			}
			switch {
			case innerKind == reflect.String:
				w.WriteString(fmt.Sprintf("s.String(y, %s)\n", compact))
			case innerKind == reflect.Struct:
				pending = append(pending, pendingData{Name: _typ.Elem().Name(), Type: _typ.Elem()})
				w.WriteString("s.BeginBlock(\"\"); s.Serialize(y); s.EndBlock();\n")
			case innerKind == reflect.Slice && _typ.Elem().Elem().Kind() == reflect.Uint8: // json.RawMessage
				w.WriteString("s.String(string(y), false)\n")
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
			w.WriteString("for _, k := range serialize.Keys(x." + name + ") {\n")
			w.WriteString("v := x." + name + "[k]\n")
			innerKind := _typ.Elem().Kind()
			switch {
			case innerKind == reflect.String:
				w.WriteString("s.KeyString(k, string(v))\n")
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
			conditional := false
			if _typ.Implements(reflect.TypeOf((*isEmpty)(nil)).Elem()) {
				w.WriteString("if !x." + name + ".IsEmpty() {\n")
				conditional = true
			}
			if _typ.Implements(reflect.TypeOf((*isHook)(nil)).Elem()) {
				// Hooked types know how to serialize themselves
				w.WriteString("x." + name + ".SerializeHook(\"" + jsonName + "\", s)\n")
			} else {
				pending = append(pending, pendingData{Name: _typ.Name(), Type: _typ})
				if !f.Anonymous {
					w.WriteString("s.BeginBlock(\"" + jsonName + "\")\n")
				}
				w.WriteString("x." + name + ".Serialize(s)\n")
				if !f.Anonymous {
					w.WriteString("s.EndBlock()\n")
				}
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

func generateGo() {

	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	g := &generator{
		visited: make(map[string]struct{}),
	}
	w.WriteString("package fiware\n\n")
	w.WriteString("import (\n\"github.com/warpcomdev/fiware/internal/serialize\"\n)\n\n")
	w.WriteString("// Autogenerated file - DO NOT EDIT\n\n")
	g.serialize(reflect.TypeOf(fiware.Manifest{}), w)
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

// Escribe la función "serialize" de una estructura
func (g *generator) serializeCue(t reflect.Type, w io.StringWriter, anonymous, tag bool) {
	textTag := "\n"
	if !anonymous {
		w.WriteString("\n#" + t.Name() + ": {\n")
	}
	if tag {
		textTag = " @anonymous(" + t.Name() + ")\n"
	}
	type pendingData struct {
		Name string
		Type reflect.Type
	}
	// Use ordered list instead of map so that types as generated
	// always in the same order they are met.
	pending := make([]pendingData, 0, 16)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		_typ := f.Type
		tags := f.Tag
		// Reuse json tags for field names
		jsonTags := strings.Split(tags.Get("json"), ",")
		if jsonTags[0] == "-" {
			continue
		}
		jsonName := jsonTags[0]
		if strings.HasPrefix(jsonName, "__") {
			// Skip the field, cue will not support it
			continue
		}
		omitempty := ""
		if len(jsonTags) > 1 && jsonTags[1] == "omitempty" {
			omitempty = "?"
		}
		switch {
		case _typ.Kind() == reflect.String:
			w.WriteString(jsonName + omitempty + ": string" + textTag)
		case _typ.Kind() == reflect.Int:
			w.WriteString(jsonName + omitempty + ": int" + textTag)
		case _typ.Kind() == reflect.Float64:
			w.WriteString(jsonName + omitempty + ": float" + textTag)
		case _typ.Kind() == reflect.Bool:
			w.WriteString(jsonName + omitempty + ": bool" + textTag)
		case _typ.Kind() == reflect.Slice && _typ.Elem().Kind() == reflect.Uint8: // json.RawMessage
			w.WriteString(jsonName + omitempty + ": #Json" + textTag)
		case _typ.Kind() == reflect.Slice: // other slices
			innerKind := _typ.Elem().Kind()
			switch {
			case innerKind == reflect.String:
				w.WriteString(jsonName + omitempty + ": [...string]" + textTag)
			case innerKind == reflect.Struct:
				innerName := _typ.Elem().Name()
				w.WriteString(jsonName + omitempty + ": [...#" + innerName + "]" + textTag)
				pending = append(pending, pendingData{Name: innerName, Type: _typ.Elem()})
			case innerKind == reflect.Slice && _typ.Elem().Elem().Kind() == reflect.Uint8: // json.RawMessage
				w.WriteString(jsonName + omitempty + ": [..._]" + textTag)
			default:
				log.Fatalf("Unknown slice type: %s", innerKind)
			}
		case _typ.Kind() == reflect.Map: // other maps
			innerKind := _typ.Elem().Kind()
			switch {
			case innerKind == reflect.String:
				w.WriteString(jsonName + omitempty + ": [string]: string" + textTag)
			case innerKind == reflect.Slice: // json.RawMessage
				w.WriteString(jsonName + omitempty + ": [string]: #Json" + textTag)
			case innerKind == reflect.Struct:
				innerName := _typ.Elem().Name()
				w.WriteString(jsonName + omitempty + ": [string]: #" + innerName + textTag)
				pending = append(pending, pendingData{Name: innerName, Type: _typ.Elem()})
			default:
				log.Fatalf("unknown map type: %s", innerKind)
			}
		case _typ.Kind() == reflect.Struct:
			innerName := _typ.Name()
			if f.Anonymous {
				g.serializeCue(_typ, w, true, true)
			} else {
				switch innerName {
				case "OptionalBool":
					w.WriteString(jsonName + "?: bool\n")
				default:
					w.WriteString(jsonName + omitempty + ": #" + innerName + "\n")
					pending = append(pending, pendingData{Name: innerName, Type: _typ})
				}
			}
		}
	}
	if !anonymous {
		w.WriteString("}\n")
	}
	for _, visiting := range pending {
		if _, ok := g.visited[visiting.Name]; !ok {
			g.visited[visiting.Name] = struct{}{}
			g.serializeCue(visiting.Type, w, false, false)
		}
	}
}

func generateCue() {

	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	g := &generator{
		visited: make(map[string]struct{}),
	}
	w.WriteString("// Autogenerated file - DO NOT EDIT\n\n")
	g.serializeCue(reflect.TypeOf(fiware.Manifest{}), w, true, false)
	w.WriteString("\n#Json: _ // cuaquier cosa...\n")
	w.Flush()

	rawCode, filename, result := buffer.Bytes(), "serializations.cue", 0
	fmtCode, err := cueformat.Source(rawCode)
	if err != nil {
		log.Printf("Failed to format code code: %v\n", err)
		fmtCode, filename, result = rawCode, "serializations.cue.fail", -1
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

func main() {
	generateGo()
	generateCue()
}
