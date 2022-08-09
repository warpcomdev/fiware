package urbo

import (
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
func (u *Urbo) slugResource(client *http.Client, headers http.Header, apiPath string, buffer interface{}) error {
	path, err := u.URL.Parse(apiPath)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	path.RawQuery = query.Encode()
	if err := keystone.GetJSON(client, headers, path, buffer, true); err != nil {
		return err
	}
	return nil
}

// Rules reads the list of rules from Perseo
func (u *Urbo) Panels(client *http.Client, headers http.Header) (map[string]fiware.UrboPanel, error) {
	var response []fiware.UrboPanel
	if err := u.slugResource(client, headers, "/api/panels", &response); err != nil {
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
	if err := u.slugResource(client, headers, "/api/verticals", &response); err != nil {
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
	path.RawQuery = query.Encode()
	var buffer json.RawMessage
	if err := keystone.GetJSON(client, headers, path, &buffer, true); err != nil {
		return nil, err
	}
	return buffer, nil
}
