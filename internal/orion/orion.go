package orion

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

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
	response []fiware.Subscription
}

// GetBuffer implements Paginator
func (p *suscriptionPaginator) GetBuffer() interface{} {
	var buffer []fiware.Subscription
	return &buffer
}

// PutBuffer implements Paginator
func (p *suscriptionPaginator) PutBuffer(buf interface{}) int {
	buffer := buf.(*[]fiware.Subscription)
	p.response = append(p.response, *buffer...)
	count := len(*buffer)
	return count
}

type registrationPaginator struct {
	response []fiware.Registration
}

// GetBuffer implements Paginator
func (p *registrationPaginator) GetBuffer() interface{} {
	var buffer []fiware.Registration
	return &buffer
}

// PutBuffer implements Paginator
func (p *registrationPaginator) PutBuffer(buf interface{}) int {
	buffer := buf.(*[]fiware.Registration)
	p.response = append(p.response, *buffer...)
	count := len(*buffer)
	return count
}

func simplifyEndpoint(ep string) string {
	var buf strings.Builder
	var last rune
	for i, r := range ep {
		if r == '/' || r == '?' || r == '&' {
			r = ':'
		}
		if r != last || r != ':' || i == 0 {
			buf.WriteRune(r)
			last = r
		}
	}
	return buf.String()
}

// Subscriptions reads the list of suscriptions from the Context Broker
func (o *Orion) Subscriptions(client keystone.HTTPClient, headers http.Header, notifEndpoints map[string]string) ([]fiware.Subscription, error) {
	path, err := o.URL.Parse("v2/subscriptions")
	if err != nil {
		return nil, err
	}
	var pages suscriptionPaginator
	if err := keystone.GetPaginatedJSON(client, headers, path, &pages, o.AllowUnknownFields); err != nil {
		return nil, err
	}
	reverseEndpoints := make(map[string]string, len(notifEndpoints))
	for k, v := range notifEndpoints {
		reverseEndpoints[v] = k
	}
	simplify := func(empty bool, url *string) {
		if !empty {
			var (
				simplified string
				hit        bool
			)
			if simplified, hit = reverseEndpoints[*url]; !hit {
				simplified = simplifyEndpoint(*url)
				reverseEndpoints[*url] = simplified
				notifEndpoints[simplified] = *url
			}
			*url = simplified
		}
	}
	for idx, sub := range pages.response {
		simplify(sub.Notification.HTTP.IsEmpty(), &sub.Notification.HTTP.URL)
		simplify(sub.Notification.HTTPCustom.IsEmpty(), &sub.Notification.HTTPCustom.URL)
		simplify(sub.Notification.MQTT.IsEmpty(), &sub.Notification.MQTT.URL)
		simplify(sub.Notification.MQTTCustom.IsEmpty(), &sub.Notification.MQTTCustom.URL)
		pages.response[idx] = sub
	}
	return pages.response, nil
}

// Suscriptions reads the list of suscriptions from the Context Broker
func (o *Orion) Registrations(client keystone.HTTPClient, headers http.Header) ([]fiware.Registration, error) {
	path, err := o.URL.Parse("v2/registrations")
	if err != nil {
		return nil, err
	}
	var pages registrationPaginator
	if err := keystone.GetPaginatedJSON(client, headers, path, &pages, o.AllowUnknownFields); err != nil {
		return nil, err
	}
	return pages.response, nil
}

// PostSuscriptions posts a list of suscriptions to orion
func (o *Orion) PostSuscriptions(client keystone.HTTPClient, headers http.Header, subs []fiware.Subscription, ep map[string]string, useDescription bool) error {
	var errList error
	if useDescription {
		epCopy := make(map[string]string, len(ep))
		for k, v := range ep {
			epCopy[k] = v
		}
		// Check there is not a subscription with the same description
		descId := make(map[string]string)
		allSubs, err := o.Subscriptions(client, headers, epCopy)
		if err != nil {
			return err
		}
		for _, sub := range allSubs {
			if sub.Description != "" {
				descId[sub.Description] = sub.ID
			}
		}
		for _, sub := range subs {
			if sub.Description != "" {
				if _, ok := descId[sub.Description]; ok {
					err := fmt.Errorf("subscription with description %s already exists", sub.Description)
					errList = multierror.Append(errList, err)
				}
			}
		}
		if errList != nil {
			return errList
		}
	}
	for _, sub := range subs {
		sub.SubscriptionStatus = fiware.SubscriptionStatus{}
		sub.Notification.NotificationStatus = fiware.NotificationStatus{}
		path, err := o.URL.Parse("v2/subscriptions")
		if err != nil {
			return err
		}
		sub, err = sub.UpdateEndpoint(ep)
		if err != nil {
			errList = multierror.Append(errList, err)
		} else {
			if _, _, err := keystone.Update(client, http.MethodPost, headers, path, sub); err != nil {
				errList = multierror.Append(errList, err)
			}
		}
	}
	return errList
}

// DeleteSuscriptions deletes a list of suscriptions from Orion
func (o *Orion) DeleteSuscriptions(client keystone.HTTPClient, headers http.Header, subs []fiware.Subscription, useDescription bool) error {
	var errList error
	byDescription := make(map[string]struct{})
	for _, sub := range subs {
		if sub.ID == "" {
			if !useDescription || sub.Description == "" {
				return errors.New("All suscriptions must have an ID")
			} else {
				if useDescription {
					byDescription[sub.Description] = struct{}{}
				}
			}
		}
		path, err := o.URL.Parse(fmt.Sprintf("v2/subscriptions/%s", sub.ID))
		if err != nil {
			return err
		}
		if _, err := keystone.Query(client, http.MethodDelete, headers, path, nil, false); err != nil {
			var netErr keystone.NetError
			if useDescription && errors.As(err, &netErr) {
				if netErr.StatusCode == 404 {
					byDescription[sub.Description] = struct{}{}
				} else {
					errList = multierror.Append(errList, err)
				}
			} else {
				errList = multierror.Append(errList, err)
			}
		}
	}
	if len(byDescription) <= 0 {
		return errList
	}
	// If there are some subscriptions we have to remove by description,
	// collect the current subscriptions and try to match them
	epCopy := make(map[string]string)
	allSubs, err := o.Subscriptions(client, headers, epCopy)
	if err != nil {
		errList = multierror.Append(errList, err)
		return errList
	}
	for _, sub := range allSubs {
		if sub.Description != "" {
			if _, ok := byDescription[sub.Description]; ok {
				path, err := o.URL.Parse(fmt.Sprintf("v2/subscriptions/%s", sub.ID))
				if err != nil {
					return err
				}
				if _, err := keystone.Query(client, http.MethodDelete, headers, path, nil, false); err != nil {
					errList = multierror.Append(errList, err)
				}
			}
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
			index := -1
			for idx, detail := range newType.Attrs {
				if detail.Name == attr {
					found = true
					index = idx
					break
				}
			}
			if !found {
				newType.Attrs = append(newType.Attrs, fiware.Attribute{
					Name:      attr,
					Type:      val.Type,
					Value:     val.Value,
					Metadatas: val.Metadatas,
				})
				index = len(newType.Attrs) - 1
			}
			if val.Type == "TextUnrestricted" {
				newType.Attrs[index].Type = "TextUnrestricted"
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
}

// GetBuffer implements Paginator
func (p *entityPaginator) GetBuffer() interface{} {
	buffer := make([]Entity, 0, batchSize)
	return &buffer
}

// PutBuffer implements Paginator
func (p *entityPaginator) PutBuffer(buf interface{}) int {
	buffer := buf.(*[]Entity)
	p.response = append(p.response, *buffer...)
	count := len(*buffer)
	ids := sort.StringSlice(make([]string, 0, len(*buffer)))
	for _, entity := range *buffer {
		ids = append(ids, entity.ID())
	}
	ids.Sort()
	return count
}

// Entities reads the list of entities from the Context Broker
func (o *Orion) Entities(client keystone.HTTPClient, headers http.Header, idPattern string, entityType string, simpleQuery string) ([]fiware.EntityType, []fiware.Entity, error) {
	path, err := o.URL.Parse("v2/entities")
	if err != nil {
		return nil, nil, err
	}
	// If filtered, add parameters
	if idPattern != "" || entityType != "" {
		values := path.Query()
		if idPattern != "" {
			values.Add("idPattern", idPattern)
		}
		if entityType != "" {
			values.Add("type", entityType)
		}
		if simpleQuery != "" {
			values.Add("q", simpleQuery)
		}
		path.RawQuery = values.Encode()
	}
	var pages entityPaginator
	if err := keystone.GetPaginatedJSON(client, headers, path, &pages, o.AllowUnknownFields); err != nil {
		return nil, nil, err
	}
	t, e := Split(pages.response)
	return t, e, nil
}

// UpdateEntities updates a list of entities
func (o *Orion) UpdateEntities(client keystone.HTTPClient, headers http.Header, ents []Entity) error {
	for base := 0; base < len(ents); base += batchSize {
		if base > 0 {
			// Wait for a timeout, the CB is this slow.
			<-time.After(3 * time.Second)
		}
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
func (o *Orion) DeleteEntities(client keystone.HTTPClient, headers http.Header, ents []fiware.Entity) error {
	type deleteEntity struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	var lastError error
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
			lastError = err // keep trying!
		}
	}
	return lastError
}
