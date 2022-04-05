package orion

import (
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

// Suscriptions reads the list of suscriptions from the Context Broker
func (o *Orion) Suscriptions(client *http.Client, headers http.Header) ([]fiware.Suscription, error) {
	path, err := o.URL.Parse("v2/subscriptions")
	if err != nil {
		return nil, err
	}
	var response []fiware.Suscription
	if err := keystone.GetJSON(client, headers, path, &response, o.AllowUnknownFields); err != nil {
		return nil, err
	}
	return response, nil
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
		if _, err := keystone.Update(client, http.MethodPost, headers, path, sub); err != nil {
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

// DeleteSuscriptions deletes a list of suscriptions from Orion
func (o *Orion) DeleteEntities(client *http.Client, headers http.Header, ents []fiware.Entity) error {
	type deleteEntity struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	req := struct {
		ActionType string         `json:"actionType"`
		Entities   []deleteEntity `json:"entities"`
	}{
		ActionType: "delete",
		Entities:   make([]deleteEntity, 0, len(ents)),
	}
	for _, e := range ents {
		req.Entities = append(req.Entities, deleteEntity{
			ID:   e.ID,
			Type: e.Type,
		})
	}
	if len(req.Entities) <= 0 {
		return nil
	}
	path, err := o.URL.Parse("v2/op/update")
	if err != nil {
		return err
	}
	if _, err := keystone.Update(client, http.MethodPost, headers, path, req); err != nil {
		return err
	}
	return nil
}
