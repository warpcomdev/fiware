package urbo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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
func (u *Urbo) slugResource(client *http.Client, headers http.Header, apiPath string) (map[string]json.RawMessage, error) {
	path, err := u.URL.Parse(apiPath)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	path.RawQuery = query.Encode()
	var response []struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		Slug        string `json:"slug,omitempty"`
	}
	if err := keystone.GetJSON(client, headers, path, &response, true); err != nil {
		return nil, err
	}
	panels := make(map[string]json.RawMessage)
	for _, p := range response {
		panels[p.Slug] = []byte(fmt.Sprintf("%q", p.Name))
	}
	return panels, nil
}

// Rules reads the list of rules from Perseo
func (u *Urbo) Panels(client *http.Client, headers http.Header) (map[string]json.RawMessage, error) {
	return u.slugResource(client, headers, "/api/panels")
}

// Rules reads the list of rules from Perseo
func (u *Urbo) Verticals(client *http.Client, headers http.Header) (map[string]json.RawMessage, error) {
	return u.slugResource(client, headers, "/api/verticals")
}
