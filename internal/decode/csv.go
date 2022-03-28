package decode

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/warpcomdev/fiware"
)

type header struct {
	Name     string
	Type     string
	IsNumber bool
	IsJson   bool
}

// skips possible BOM at beginning of utf-8 file.
// See http://www.unicode.org/faq/utf_bom.html#BOM
func skipBOM(filename string) (io.ReadCloser, error) {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	br := bufio.NewReader(fd)
	r, _, err := br.ReadRune()
	if err != nil {
		fd.Close()
		return nil, err
	}
	if r != '\uFEFF' {
		br.UnreadRune() // Not a BOM -- put the rune back
	}
	type readCloser struct {
		io.Reader
		io.Closer
	}
	return readCloser{Reader: br, Closer: fd}, nil
}

// Parse CSV header
func parseHeader(headers []string) []header {
	result := make([]header, 0, len(headers))
	for _, item := range headers {
		item_name, item_type := item, ""
		if !strings.Contains(item, "<") {
			item_name = strings.TrimSpace(item_name)
		} else {
			parts := strings.SplitN(item, "<", 2)
			item_name = strings.TrimSpace(parts[0])
			item_type = strings.TrimSpace(strings.SplitN(parts[1], ">", 2)[0])
		}
		h := header{
			Name: item_name,
			Type: item_type,
		}
		lower := strings.ToLower(h.Type)
		switch {
		case lower == "number":
			h.IsNumber = true
		case strings.Contains(lower, "json"):
			h.IsJson = true
		}
		result = append(result, h)
	}
	return result
}

// Reading from CSV, there might be entities with the same
// attribute name but different attribute type, if someone
// has changed the type by hand.
// so we use both the name and the type as keys to identify
// when an attribute has already appeared.
type setKey struct {
	Name string
	Type string
}

type entityWithSet struct {
	fiware.EntityType
	set map[setKey]struct{}
}

func get_models_csv(filename string) ([]fiware.EntityType, []fiware.Entity) {
	infile, err := skipBOM(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer infile.Close()
	reader := csv.NewReader(infile)
	reader.ReuseRecord = true // We always copy strings to bytes
	first, err := reader.Read()
	if err != nil {
		log.Fatalf("Failed to read CSV header: %v", err)
	}
	headers, row := parseHeader(first), 1
	if len(headers) < 2 || strings.ToLower(headers[0].Name) != "entityid" || strings.ToLower(headers[1].Name) != "entitytype" {
		log.Fatalf("Headers must begin with entityID, entityType, not '%s', '%s'", headers[0].Name, headers[1].Name)
	}
	headers = headers[2:]
	mixedEntities := make([]fiware.EntityType, 0, 64)
	for {
		next, err := reader.Read()
		row += 1
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Fatalf("Failed to read CSV row %d: %v", row, err)
			}
			break
		}
		if len(next) < 2 {
			continue
		}
		current := fiware.EntityType{
			ID:    next[0],
			Type:  next[1],
			Attrs: make([]fiware.Attribute, len(headers)),
		}
		for index, col := range next[2:] {
			if index > len(headers) {
				break
			}
			col = strings.TrimSpace(col)
			if col == "" || col == "\"\"" {
				continue
			}
			h := headers[index]
			if h.Type == "none" || h.Type == "None" {
				continue
			}
			switch {
			case h.IsNumber:
				current.Attrs[index] = importNumber(col)
			case h.IsJson:
				current.Attrs[index] = importJson(col)
			default:
				current.Attrs[index] = importOther(col)
			}
		}
		mixedEntities = append(mixedEntities, current)
	}
	// Para cada tipo, acumulo todos los atributos de las
	// entidades de ese tipo
	bytype := make(map[string]entityWithSet)
	for _, entity := range mixedEntities {
		ref, changed := bytype[entity.Type], false
		if ref.Type == "" {
			ref.ID = entity.ID
			ref.Type = entity.Type
			ref.Attrs = make([]fiware.Attribute, 0, len(headers))
			ref.set = make(map[setKey]struct{})
			changed = true
		}
		for index, attr := range entity.Attrs {
			if len(attr.Value) <= 0 { // Skip attributes the entity doesn't have
				continue
			}
			h := headers[index]
			attr.Name = h.Name
			attr.Type = h.Type
			key := setKey{Name: attr.Name, Type: attr.Type}
			if _, ok := ref.set[key]; !ok {
				// First time this attribute appears for this entity type
				ref.Attrs = append(ref.Attrs, attr)
				ref.set[key] = struct{}{}
				changed = true
			}
		}
		if changed {
			bytype[entity.Type] = ref
		}
	}
	// Turn the entity map into a list, sorted by natity type
	entityTypes := make([]fiware.EntityType, 0, len(bytype))
	for _, e := range bytype {
		entityTypes = append(entityTypes, e.EntityType)
	}
	sort.Slice(entityTypes, func(i, j int) bool {
		return strings.Compare(entityTypes[i].Type, entityTypes[j].Type) < 0
	})
	// Extract the streamlined entities from the mix
	entities := make([]fiware.Entity, 0, len(mixedEntities))
	for _, entity := range mixedEntities {
		values := make(map[string]json.RawMessage)
		metadatas := make(map[string]json.RawMessage)
		for index, attr := range entity.Attrs {
			h := headers[index]
			if attr.Value != nil && len(attr.Value) > 0 {
				values[h.Name] = attr.Value
				if attr.Metadatas != nil && len(attr.Metadatas) > 0 {
					metadatas[h.Name] = attr.Metadatas
				}
			}
		}
		curr := fiware.Entity{
			ID:   entity.ID,
			Type: entity.Type,
		}
		if len(values) > 0 {
			curr.Attrs = values
		}
		if len(metadatas) > 0 {
			curr.MetaDatas = metadatas
		}
		entities = append(entities, curr)
	}
	// sort final entities by type
	sort.Slice(entities, func(i, j int) bool {
		return strings.Compare(entities[i].Type, entities[j].Type) < 0
	})
	return entityTypes, entities
}
