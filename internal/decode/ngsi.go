package decode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/warpcomdev/fiware"
)

// Tries to guess the separator used in a string
func findsep(multistr string) string {
	// Check if the string is a sequence of words separated by '/'
	parts := strings.Split(multistr, "/")
	if len(parts) > 1 {
		for _, part := range parts {
			// If any part is not a single word, assume comma-separated
			if words := strings.Fields(strings.TrimSpace(part)); len(words) > 1 {
				return ","
			}
		}
		return "/"
	}
	// If no slashes in it, assume comma-separated
	return ","
}

var wifi_repeats = regexp.MustCompile(`^(?P<pre>[^{]+){(?P<mid>[^}]+)}(?P<post>.*)$`)

// Generate attribute from README line
func from_line(longtermIndex int, line string) []fiware.Attribute {
	// log.Printf("Parsing line %s", line)
	// Warning: WiFi vertical uses field names like
	// dlBandwidth{User\|Device} to summarize two lines in one
	if strings.HasPrefix(line, "|") {
		line = strings.TrimSpace(line[1:])
	}
	line = strings.ReplaceAll(line, "\\|", "__PLACEHOLDER_SEPARATOR__")
	fields := strings.Split(line, "|")
	for index, part := range fields {
		fields[index] = strings.TrimSpace(strings.ReplaceAll(part, "__PLACEHOLDER_SEPARATOR__", "|"))
	}
	name := fields[0]
	_typ := fields[1]
	desc := make([]string, 0, len(fields))
	for _, item := range fields[2:] {
		item = strings.TrimSpace(item)
		if item != "" && item != "-" {
			desc = append(desc, item)
		}
	}
	if len(desc) <= 0 {
		desc = nil
	}
	if strings.HasPrefix(name, "[") { // for commands
		if index := strings.Index(name[1:], "]"); index > 0 {
			// strings.Index was passed name[1:], so we have to add 1
			name = name[1:(index + 1)]
		}
	}
	/*if strings.HasPrefix(strings.ToLower(_typ), "command") {
		log.Printf("Skipping %s %s (not added to models or suscriptions)", _typ, name)
		return nil
	}*/
	//log.Printf("Parsing attribute %s type %s", name, _typ)
	seek := []string{
		"Ejemplos", "ejemplos", "Ejemplo", "ejemplo", "Valores", "valores", "Valor", "valor",
		"Examples", "examples", "Example", "example", "Values", "values", "Value", "value",
	}
	seps := map[string]bool{":": true, "=": true}
	quot := map[string]string{"\"": "\"", "`": "`", "'": "'", "[": "]"}
	singletonKey, simulated := false, false
	var text []string
	// Check if we have a longterm column
	longterm := fiware.LongtermNone
	var longtermOptions []string
	if longtermIndex >= 2 {
		after := strings.TrimSpace(fields[longtermIndex])
		afterLower := strings.ToLower(after)
		if strings.HasPrefix(afterLower, "modal") {
			longterm = fiware.LongtermModal
		}
		if strings.HasPrefix(afterLower, "gauge") {
			longterm = fiware.LongtermGauge
		}
		if strings.HasPrefix(afterLower, "counter") {
			longterm = fiware.LongtermCounter
		}
		if strings.HasPrefix(afterLower, "dimension") {
			longterm = fiware.LongtermDimension
		}
		if strings.HasPrefix(afterLower, "enum") {
			longterm = fiware.LongtermEnum
			options := strings.Split(strings.TrimSpace(strings.SplitN(after, " ", 2)[1]), ",")
			longtermOptions = make([]string, 0, len(options))
			for _, option := range options {
				if strings.HasPrefix(option, "'") || strings.HasPrefix(option, "\"") {
					option = option[1 : len(option)-1]
				}
				longtermOptions = append(longtermOptions, strings.TrimSpace(option))
			}
		}
	}
	// Check other columns for samples
	for _, other := range fields[2:] {
		otherLower := strings.ToLower(other)
		if strings.Contains(otherLower, "singleton") {
			singletonKey = true
		}
		if strings.Contains(otherLower, "simulación") || strings.Contains(otherLower, "simulacion") {
			simulated = true
		}
		for _, substr := range seek {
			index := strings.Index(other, substr)
			if index >= 0 {
				remaining := strings.TrimSpace(other[(index + len(substr)):])
				if len(remaining) > 0 {
					first := remaining[:1]
					// Must match seek + sep
					if len(remaining) <= 0 || !seps[first] {
						continue
					}
					remaining = strings.TrimSpace(remaining[1:])
					first = remaining[:1]
					if closequot := quot[first]; closequot != "" {
						if end := strings.Index(remaining[1:], closequot); end >= 0 {
							remaining = strings.TrimSpace(remaining[1:(end + 1)])
						}
					}
					// If it is several examples or values, take the first
					if strings.HasSuffix(substr, "s") {
						sep := findsep(remaining)
						text = strings.Split(remaining, sep)
						for index, part := range text {
							text[index] = strings.TrimSpace(part)
						}
					} else {
						text = []string{strings.TrimSpace(remaining)}
					}
					// log.Printf("Found %s: %s", substr, text)
					break
				}
			}
		}
		// Do NOT break. Examples in later cols override
		// examples in prev cols
		//if len(text) > 0:
		//    break
	}
	// Turn value into the proper type
	value := make([]fiware.Attribute, 0, len(text))
	lower := strings.ToLower(_typ)
	for _, v := range text {
		v = strings.TrimSpace(v)
		switch {
		case lower == "number":
			if v == "" {
				value = append(value, fiware.Attribute{Value: []byte("null")})
			} else {
				value = append(value, importNumber(v))
			}
		case strings.HasPrefix(lower, "list "):
			fallthrough
		case strings.Contains(lower, "json"):
			if v == "" {
				value = append(value, fiware.Attribute{Value: []byte("null")})
			} else {
				value = append(value, importJson(v))
			}
		default:
			value = append(value, importOther(v))
		}
	}
	// Finally, check for wifi-style repeats
	matches := wifi_repeats.FindStringSubmatch(name)
	if len(matches) <= 0 {
		attrib := fiware.Attribute{
			Name:            name,
			Type:            _typ,
			Description:     desc,
			SingletonKey:    singletonKey,
			Simulated:       simulated,
			Longterm:        longterm,
			LongtermOptions: longtermOptions,
		}
		if len(value) > 0 {
			attrib.Value = value[0].Value
			attrib.Metadatas = value[0].Metadatas
		}
		return []fiware.Attribute{attrib}
	}
	log.Printf("Found infixed attribute %s", name)
	prefix := strings.TrimSpace(matches[1])
	suffix := strings.TrimSpace(matches[3])
	infixes := make([]string, 0, 16)
	for _, part := range strings.Split(matches[2], "|") {
		infixes = append(infixes, strings.TrimSpace(part))
	}
	vals := len(value)
	result := make([]fiware.Attribute, 0, len(infixes))
	for index, infix := range infixes {
		attrib := fiware.Attribute{
			Name:            fmt.Sprintf("%s%s%s", prefix, infix, suffix),
			Type:            _typ,
			Description:     desc,
			SingletonKey:    singletonKey,
			Simulated:       simulated,
			Longterm:        longterm,
			LongtermOptions: longtermOptions,
		}
		if vals > 0 {
			attrib.Value = value[index%vals].Value
			attrib.Metadatas = value[index%vals].Metadatas
		}
		result = append(result, attrib)
	}
	return result
}

// Builds model from list of lines
func from_lines(longtermIndex int, lines []string) fiware.EntityType {
	model := fiware.EntityType{ID: "", Type: "", Attrs: make([]fiware.Attribute, 0, 16)}
	visited := make(map[string]struct{})
	for _, line := range lines {
		for _, attrib := range from_line(longtermIndex, line) {
			name := strings.ToLower(attrib.Name)
			switch {
			case name == "id":
				log.Printf("Found entity ID = %s", attrib.Value)
				if err := json.Unmarshal(attrib.Value, &(model.ID)); err != nil {
					log.Printf("no se pudo decodificar el ID de entidad")
				}
			case name == "tipo" || name == "type":
				log.Printf("Found entity type = %s", attrib.Value)
				if err := json.Unmarshal(attrib.Value, &(model.Type)); err != nil {
					log.Printf("no se pudo decodificar el tipo de entidad")
				}
			default:
				if _, existing := visited[name]; existing {
					log.Printf("El atributo %s de la entidad %s está repetido", attrib.Name, model.Type)
				} else {
					visited[name] = struct{}{}
					model.Attrs = append(model.Attrs, attrib)
				}
			}
		}
	}
	return model
}

// Get a list of models from README file.
// Expects tables with a header like the following:
// |Atributo|Tipo|Descripción|Información adicional|Ud|Rango|
// Recognizes these particular pieces of information:
// - atributo = id: Entity ID
// - atributo = tipo|type: entity type
// - tipo: Text, TextUnrestricted, Number, Reference, geo:json, geox:json ...
// - Any other column: "Ejemplo:", "Ejemplo=", "Valor:", "Valor=",...
func NGSI(filename string) ([]fiware.EntityType, []fiware.Entity) {
	models := make([]fiware.EntityType, 0, 16)
	entities := make([]fiware.Entity, 0, 16)
	latest := make([]string, 0, 256)
	inside := false
	infile, err := SkipBOM(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer infile.Close()
	scanner := bufio.NewScanner(infile)
	mustPipe := true    // true if table lines MUST start with "|"
	longtermIndex := -1 // columna que contiene el tipo de longterm
	done := false
	for !done {
		var line string
		if scanner.Scan() {
			line = strings.TrimSpace(scanner.Text())
		} else {
			// Scan one empty line after scanner.Scan() returns false,
			// to detect model end if the model is the last thing in
			// the file and there is no ending empty line.
			line = ""
			done = true
		}
		if inside {
			// Empty line is the end of a block
			if line == "" {
				model := from_lines(longtermIndex, latest)
				log.Printf("Finished processing model %s", model.Type)
				models = append(models, model)
				inside = false
				longtermIndex = -1
				latest = latest[:0]
				// Create entity too, to be able to populate CSV from NGSI
				entity := fiware.Entity{
					ID:        model.ID,
					Type:      model.Type,
					Attrs:     make(map[string]json.RawMessage),
					MetaDatas: make(map[string]json.RawMessage),
				}
				for _, attr := range model.Attrs {
					if attr.Value != nil && len(attr.Value) > 0 {
						entity.Attrs[attr.Name] = attr.Value
					}
					if attr.Metadatas != nil && len(attr.Metadatas) > 0 {
						entity.MetaDatas[attr.Name] = attr.Metadatas
					}
				}
				entities = append(entities, entity)
			} else {
				switch {
				case !mustPipe && strings.Contains(line, "|"):
					// Many verticals use an alternate format for markdown table
					// without the initial "|"... we have to rely on detecting a
					// "|" as a signal of table line
					fallthrough
				case strings.HasPrefix(line, "|"):
					// Line starting with "|" is another attribute
					// unless it is just an underline
					if !strings.HasPrefix(strings.TrimSpace(line[1:]), "-") {
						latest = append(latest, line)
					} else {
						log.Printf("Skipping separator line")
					}
				default:
					// Other lines are continuations from prev attrib
					log.Printf("Detected continuation line")
					index := len(latest) - 1
					if index >= 0 {
						latest[index] = latest[index] + " " + line
					}
				}
			}
		} else {
			lower := strings.TrimSpace(strings.ToLower(line))
			initialPipe := false
			if strings.HasPrefix(lower, "|") {
				lower = strings.TrimSpace(lower[1:])
				initialPipe = true
			}
			if strings.HasPrefix(lower, "atributo|") || strings.HasPrefix(lower, "atributo ") || strings.HasPrefix(lower, "attribute") || strings.HasPrefix(lower, "atrribute ") {
				log.Print("Detected model start")
				inside = true
				mustPipe = initialPipe
				// Find out which is the longterm column, if any
				longtermIndex = -1
				if strings.HasPrefix(lower, "|") {
					lower = strings.TrimSpace(lower[1:])
				}
				for idx, head := range strings.Split(lower, "|") {
					if strings.TrimSpace(head) == "longterm" {
						longtermIndex = idx
					}
				}
			}
		}
	}
	return models, entities
}
