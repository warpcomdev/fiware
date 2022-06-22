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
}

// New Urbo client instance
func New(urboURL string, username, service string) (*Urbo, error) {
	URL, err := url.Parse(fmt.Sprintf("%s", urboURL))
	if err != nil {
		return nil, err
	}
	return &Urbo{
		URL:      URL,
		Username: username,
		Service:  service,
	}, nil
}

func (u *Urbo) Login(client *http.Client, password string) (string, error) {
	loginURL, err := u.URL.Parse("/api/sso/login")
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
func (u *Urbo) Panels(client *http.Client, headers http.Header) (json.RawMessage, error) {
	path, err := u.URL.Parse("panels")
	if err != nil {
		return nil, err
	}
	var response json.RawMessage
	if err := keystone.GetJSON(client, headers, path, &response, true); err != nil {
		return nil, err
	}
	//if len(response.Error) > 0 && !bytes.Equal(response.Error, []byte("null")) {
	//	return nil, fmt.Errorf("perseo replied with error: %s", string(response.Error))
	//}
	//return response.Data, nil
	return response, nil
}
