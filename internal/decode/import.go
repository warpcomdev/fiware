package decode

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/warpcomdev/fiware"
)

// encodes a expected map[string]interface{} to JSON
func mustEncode(v interface{}) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

// Importa un valor json, que puede ser un valor json como tal,
// o un objeto con valor y metadatas.
// A veces, si el csv ha sido exportado de orion, se cuela
// algun registro en formato ngsiv2, con metadatas.
func importJson(v string) fiware.Attribute {
	var structured interface{}
	if !strings.HasPrefix(v, "{") && !strings.HasPrefix(v, "[") {
		log.Printf("supposed json type %s does not start with '{' or '[', decoding as string", v)
		return fiware.Attribute{Value: []byte(fmt.Sprintf("%q", v))}
	}
	if err := json.Unmarshal([]byte(v), &structured); err != nil {
		log.Printf("failed to decode %s because of %v, assuming it's a text placeholder", v, err)
		return fiware.Attribute{Value: []byte(fmt.Sprintf("%q", v))}
	}
	// Check if the attribute is actually a value and metadata pair from orion dump
	if d, ok := structured.(map[string]interface{}); ok {
		if v, ok := d["value"]; ok {
			if m, ok := d["metadatas"]; ok {
				return fiware.Attribute{Value: mustEncode(v), Metadatas: mustEncode(m)}
			}
		}
	}
	return fiware.Attribute{Value: mustEncode(structured)}
}

func importNumber(v string) fiware.Attribute {
	// A veces, si el csv ha sido exportado de orion, se cuela
	// algun registro en formato ngsiv2, con metadata
	if strings.HasPrefix(v, "{") {
		return importJson(v)
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Printf("supposed float type %s cannot be parsed, decoding as string", v)
		return fiware.Attribute{Value: []byte(fmt.Sprintf("%q", v))}
	}
	return fiware.Attribute{Value: []byte(strconv.FormatFloat(f, 'f', 2, 64))}
}

func importOther(v string) fiware.Attribute {
	if strings.HasPrefix(v, "{") { // Por si viene con metadata
		return importJson(v)
	}
	if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") && !strings.HasPrefix(v, "\"\"") {
		// Si ya nos han dado la cadena escapada
		return fiware.Attribute{Value: []byte(v)}
	}
	if strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") {
		if len(v) <= 2 {
			v = ""
		} else {
			v = strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return fiware.Attribute{Value: []byte(fmt.Sprintf("%q", v))}
}
