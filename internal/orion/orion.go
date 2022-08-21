package orion

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-multierror"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/keystone"
)

// Orion manages connection to the Context Broker
type Orion struct {
	URL                *url.URL
	AllowUnknownFields bool
}

const batchSize = 50

// New Orion instance
func New(orionURL string) (*Orion, error) {
	apiURL, err := url.Parse(orionURL)
	if err != nil {
		return nil, err
	}
	return &Orion{
		URL: apiURL,
	}, nil
}

type suscriptionPaginator struct {
	response []fiware.Suscription
	buffer   []fiware.Suscription
}

// GetBuffer implements Paginator
func (p *suscriptionPaginator) GetBuffer() interface{} {
	return &p.buffer
}

// PutBuffer implements Paginator
func (p *suscriptionPaginator) PutBuffer(buf interface{}) int {
	p.response = append(p.response, p.buffer...)
	count := len(p.buffer)
	p.buffer = p.buffer[:0] // Reset the buffer before next cycle
	return count
}

// Suscriptions reads the list of suscriptions from the Context Broker
func (o *Orion) Suscriptions(client *http.Client, headers http.Header) ([]fiware.Suscription, error) {
	path, err := o.URL.Parse("v2/subscriptions")
	if err != nil {
		return nil, err
	}
	var pages suscriptionPaginator
	if err := keystone.GetPaginatedJSON(client, headers, path, &pages, o.AllowUnknownFields); err != nil {
		return nil, err
	}
	return pages.response, nil
}

// PostSuscriptions posts a list of suscriptions to orion
func (o *Orion) PostSuscriptions(client *http.Client, headers http.Header, subs []fiware.Suscription) error {
	var errList error
	for _, sub := range subs {
		sub.SuscriptionStatus = fiware.SuscriptionStatus{}
		sub.Notification.NotificationStatus = fiware.NotificationStatus{}
		path, err := o.URL.Parse("v2/subscriptions")
		if err != nil {
			return err
		}
		if _, _, err := keystone.Update(client, http.MethodPost, headers, path, sub); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}

// DeleteSuscriptions deletes a list of suscriptions from Orion
func (o *Orion) DeleteSuscriptions(client *http.Client, headers http.Header, subs []fiware.Suscription) error {
	var errList error
	for _, sub := range subs {
		if sub.ID == "" {
			return errors.New("All suscriptions must have an ID")
		}
		path, err := o.URL.Parse(fmt.Sprintf("v2/subscriptions/%s", sub.ID))
		if err != nil {
			return err
		}
		if err := keystone.Query(client, http.MethodDelete, headers, path, nil, false); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}

// Entity representa una entidad tal como la ve la API
type Entity map[string]json.RawMessage

func (e Entity) ID() string {
	var id string
	json.Unmarshal(e["id"], &id)
	return id
}

func (e Entity) Type() string {
	var t string
	json.Unmarshal(e["type"], &t)
	return t
}

func (e Entity) Attrs() map[string]EntityAttr {
	attrs := make(map[string]EntityAttr)
	for k, v := range e {
		if k != "id" && k != "type" {
			var attr EntityAttr
			json.Unmarshal(v, &attr)
			attrs[k] = attr
		}
	}
	return attrs
}

type EntityAttr struct {
	Type      string          `json:"type"`
	Value     json.RawMessage `json:"value"`
	Metadatas json.RawMessage `json:"metadatas,omitempty"`
}

// Merge fiware EntityType with Entity to build full Entity
func Merge(types []fiware.EntityType, values []fiware.Entity) []Entity {
	typeMap := make(map[string](map[string]fiware.Attribute), len(types))
	for _, t := range types {
		ta := make(map[string]fiware.Attribute)
		for _, a := range t.Attrs {
			ta[a.Name] = a
		}
		typeMap[t.Type] = ta
	}
	result := make([]Entity, 0, len(values))
	for _, v := range values {
		current := make(Entity)
		current["id"] = json.RawMessage(fmt.Sprintf("%q", v.ID))
		current["type"] = json.RawMessage(fmt.Sprintf("%q", v.Type))
		if tm, ok := typeMap[v.Type]; ok {
			for name, value := range v.Attrs {
				if am, ok := tm[name]; ok {
					currentAttr := EntityAttr{
						Type:  am.Type,
						Value: value,
					}
					if md, ok := v.MetaDatas[name]; ok {
						currentAttr.Metadatas = md
					} else {
						if am.Metadatas != nil {
							currentAttr.Metadatas = am.Metadatas
						}
					}
					value, _ = json.Marshal(currentAttr)
					current[name] = value
				}
			}
			result = append(result, current)
		}
	}
	return result
}

// Split a merged Entity into fiware.entityType and fiware.Entity
func Split(entities []Entity) ([]fiware.EntityType, []fiware.Entity) {
	types := make(map[string]fiware.EntityType)
	values := make([]fiware.Entity, 0, len(entities))
	for _, current := range entities {
		currID := current.ID()
		currType := current.Type()
		currAttrs := current.Attrs()
		// Add a type (if not seen already)
		newType, ok := types[currType]
		if !ok {
			newType = fiware.EntityType{
				ID:    currID,
				Type:  currType,
				Attrs: make([]fiware.Attribute, 0, len(currAttrs)),
			}
		}
		// Add an entity
		newEntity := fiware.Entity{
			ID:    currID,
			Type:  currType,
			Attrs: make(map[string]json.RawMessage),
		}
		for attr, val := range currAttrs {
			newEntity.Attrs[attr] = val.Value
			// Check all attributes are defined for the EntityType
			found := false
			for _, detail := range newType.Attrs {
				if detail.Name == attr {
					found = true
					break
				}
			}
			if !found {
				newType.Attrs = append(newType.Attrs, fiware.Attribute{
					Name:      attr,
					Value:     val.Value,
					Metadatas: val.Metadatas,
				})
			}
		}
		types[currType] = newType
		values = append(values, newEntity)
	}
	typesSlice := make([]fiware.EntityType, 0, len(types))
	for _, t := range types {
		typesSlice = append(typesSlice, t)
	}
	return typesSlice, values
}

type entityPaginator struct {
	response []Entity
	buffer   []Entity
}

// GetBuffer implements Paginator
func (p *entityPaginator) GetBuffer() interface{} {
	return &p.buffer
}

// PutBuffer implements Paginator
func (p *entityPaginator) PutBuffer(buf interface{}) int {
	p.response = append(p.response, p.buffer...)
	count := len(p.buffer)
	p.buffer = p.buffer[:0] // Reset the buffer before next cycle
	return count
}

// Entities reads the list of entities from the Context Broker
func (o *Orion) Entities(client *http.Client, headers http.Header) ([]fiware.EntityType, []fiware.Entity, error) {
	path, err := o.URL.Parse("v2/entities")
	if err != nil {
		return nil, nil, err
	}
	var pages entityPaginator
	if err := keystone.GetPaginatedJSON(client, headers, path, &pages, o.AllowUnknownFields); err != nil {
		return nil, nil, err
	}
	t, e := Split(pages.response)
	return t, e, nil
}

// UpdateEntities updates a list of entities
func (o *Orion) UpdateEntities(client *http.Client, headers http.Header, ents []Entity) error {
	for base := 0; base < len(ents); base += batchSize {
		req := struct {
			ActionType string   `json:"actionType"`
			Entities   []Entity `json:"entities"`
		}{
			ActionType: "append",
			Entities:   make([]Entity, 0, len(ents)),
		}
		top := len(ents)
		if top >= base+batchSize {
			top = base + batchSize
		}
		req.Entities = append(req.Entities, ents[base:top]...)
		path, err := o.URL.Parse("v2/op/update")
		if err != nil {
			return err
		}
		if _, _, err := keystone.Update(client, http.MethodPost, headers, path, req); err != nil {
			return err
		}
	}
	return nil
}

// DeleteEntities deletes a list of entities from Orion
func (o *Orion) DeleteEntities(client *http.Client, headers http.Header, ents []fiware.Entity) error {
	type deleteEntity struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	for base := 0; base < len(ents); base += batchSize {
		req := struct {
			ActionType string         `json:"actionType"`
			Entities   []deleteEntity `json:"entities"`
		}{
			ActionType: "delete",
			Entities:   make([]deleteEntity, 0, len(ents)),
		}
		top := len(ents)
		if top >= base+batchSize {
			top = base + batchSize
		}
		for _, e := range ents[base:top] {
			req.Entities = append(req.Entities, deleteEntity{
				ID:   e.ID,
				Type: e.Type,
			})
		}
		path, err := o.URL.Parse("v2/op/update")
		if err != nil {
			return err
		}
		if _, _, err := keystone.Update(client, http.MethodPost, headers, path, req); err != nil {
			return err
		}
	}
	return nil
}
