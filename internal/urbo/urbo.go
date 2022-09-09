package urbo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/keystone"
)

// Keystone manages Requests to the Identity Manager
type Urbo struct {
	URL               *url.URL
	Username, Service string
	ScopeService      string
}

// New Urbo client instance
func New(urboURL string, username, service, scopeService string) (*Urbo, error) {
	URL, err := url.Parse(fmt.Sprintf("%s", urboURL))
	if err != nil {
		return nil, err
	}
	return &Urbo{
		URL:          URL,
		Username:     username,
		Service:      service,
		ScopeService: scopeService,
	}, nil
}

func (u *Urbo) Login(client *http.Client, password string) (string, error) {
	loginURL, err := u.URL.Parse("/auth/sso/login")
	if err != nil {
		return "", err
	}
	payload := struct {
		Service  string `json:"service"`
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Service: u.Service, Username: u.Username, Password: password,
	}
	_, body, err := keystone.Update(client, http.MethodPost, nil, loginURL, payload)
	if err != nil {
		return "", err
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (u *Urbo) Headers(token string) (http.Header, error) {
	header := http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	return header, nil
}

// Rules reads the list of rules from Perseo
func (u *Urbo) slugResource(client *http.Client, headers http.Header, apiPath string, params map[string]string, buffer interface{}) error {
	path, err := u.URL.Parse(apiPath)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	if params != nil {
		for k, v := range params {
			query.Add(k, v)
		}
	}
	path.RawQuery = query.Encode()
	if err := keystone.GetJSON(client, headers, path, buffer, true); err != nil {
		return err
	}
	return nil
}

// Rules reads the list of rules from Perseo
func (u *Urbo) Panels(client *http.Client, headers http.Header) (map[string]fiware.UrboPanel, error) {
	var response []fiware.UrboPanel
	if err := u.slugResource(client, headers, "/api/panels", nil, &response); err != nil {
		return nil, err
	}
	panels := make(map[string]fiware.UrboPanel)
	for _, p := range response {
		panels[p.Slug] = p
	}
	return panels, nil
}

// Rules reads the list of rules from Perseo
func (u *Urbo) Verticals(client *http.Client, headers http.Header) (map[string]fiware.UrboVertical, error) {
	var response []fiware.UrboVertical
	if err := u.slugResource(client, headers, "/api/verticals", map[string]string{"shadowPanels": "true"}, &response); err != nil {
		return nil, err
	}
	verticals := make(map[string]fiware.UrboVertical)
	for _, p := range response {
		verticals[p.Slug] = p
	}
	return verticals, nil
}

// Panel reads a single panel
func (u *Urbo) Panel(client *http.Client, headers http.Header, slug string) (json.RawMessage, error) {
	path, err := u.URL.Parse(fmt.Sprintf("/api/panels/%s", slug))
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	// parameters learnt by watching urbo, not actually documented...
	// make sure the panels as returned in a form we can copy and paste them in urbo.
	query.Add("unfiltered", "true")
	query.Add("edit", "true")
	path.RawQuery = query.Encode()
	var data map[string]interface{}
	if err := keystone.GetJSON(client, headers, path, &data, true); err != nil {
		return nil, err
	}
	remove_id(data)
	buffer := bytes.Buffer{}
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// remove_id cleans a json object from "_id", and "__v" fields which are not useful
func remove_id(m interface{}) {
	if submap, ok := m.(map[string]interface{}); ok {
		for k, v := range submap {
			if k == "_id" || k == "__v" {
				delete(submap, k)
			} else {
				remove_id(v)
			}
		}
		return
	}
	if sublist, ok := m.([]interface{}); ok {
		for _, v := range sublist {
			remove_id(v)
		}
	}
}
