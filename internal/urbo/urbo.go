package urbo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

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

func (u *Urbo) Login(client keystone.HTTPClient, password string, backoff keystone.Backoff) (string, error) {
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
	var body []byte
	var retries int
	for {
		_, body, err = keystone.Update(client, http.MethodPost, nil, loginURL, payload)
		if err == nil {
			break
		}
		var netErr keystone.NetError
		if errors.As(err, &netErr) {
			if netErr.StatusCode != 500 {
				return "", err
			}
		}
		keepTrying, delay := backoff.KeepTrying(retries)
		retries += 1
		if !keepTrying {
			return "", err
		}
		<-time.After(delay)
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
func (u *Urbo) slugResource(client keystone.HTTPClient, headers http.Header, apiPath string, params map[string]string, buffer interface{}) error {
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

// Panels reads the list of panels from Urbo
func (u *Urbo) Panels(client keystone.HTTPClient, headers http.Header) (map[string]fiware.UrboPanel, error) {
	var response []fiware.UrboPanel
	if err := u.slugResource(client, headers, "/api/panels", map[string]string{"unassigned": "true"}, &response); err != nil {
		return nil, err
	}
	panels := make(map[string]fiware.UrboPanel)
	for _, p := range response {
		panels[p.Slug] = p
	}
	return panels, nil
}

type apiVertical struct {
	Panels  []fiware.UrboPanel `json:"panels,omitempty"`
	Slug    string             `json:"slug"`
	Name    string             `json:"name,omitempty"`
	I18n    json.RawMessage    `json:"i18n,omitempty"`
	Icon    string             `json:"icon,omitempty"`
	Service string             `json:"service"`
}

func slugs(panels []fiware.UrboPanel) []string {
	slugs := make([]string, 0, len(panels))
	for _, panel := range panels {
		slugs = append(slugs, panel.Slug)
	}
	return slugs
}

// GetVerticals reads the list of verticals from Urbo
func (u *Urbo) GetVerticals(client keystone.HTTPClient, headers http.Header) (map[string]fiware.Vertical, error) {
	var response []apiVertical
	if err := u.slugResource(client, headers, "/api/verticals", map[string]string{"shadowPanels": "true"}, &response); err != nil {
		return nil, err
	}
	verticals := make(map[string]fiware.Vertical)
	// Must read verticals one by one again because '/verticals'
	// does not tell regular panels from shadow panels
	for _, p := range response {
		detailed, err := u.GetVertical(client, headers, p.Slug)
		if err != nil {
			return nil, err
		}
		verticals[p.Slug] = detailed
	}
	return verticals, nil
}

// GetVerticals reads the list of verticals from Urbo
func (u *Urbo) GetVertical(client keystone.HTTPClient, headers http.Header, slug string) (fiware.Vertical, error) {
	var detailed fiware.Vertical
	if err := u.slugResource(client, headers, "/api/verticals/"+slug, map[string]string{"shadowPanels": "true"}, &detailed); err != nil {
		return detailed, err
	}
	return fiware.Vertical{
		Name:         detailed.Name,
		Slug:         detailed.Slug,
		I18n:         detailed.I18n,
		Panels:       slugs(detailed.PanelsObjects),
		ShadowPanels: slugs(detailed.ShadowPanelsObjects),
		UrboVerticalStatus: fiware.UrboVerticalStatus{
			PanelsObjects:       detailed.PanelsObjects,
			ShadowPanelsObjects: detailed.ShadowPanelsObjects,
		},
	}, nil
}

// PostVerticals reads the list of verticals from Urbo
func (u *Urbo) PostVerticals(client keystone.HTTPClient, headers http.Header, verticals map[string]fiware.Vertical) error {
	path, err := u.URL.Parse(fmt.Sprintf("/api/verticals"))
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	path.RawQuery = query.Encode()
	for _, vertical := range verticals {
		av := vertical
		vertical.UrboVerticalStatus = fiware.UrboVerticalStatus{}
		if _, _, err := keystone.PostJSON(client, headers, path, av); err != nil {
			var netErr keystone.NetError
			if errors.As(err, &netErr) {
				if netErr.StatusCode == 400 { // Panel already exists
					path, err := u.URL.Parse(fmt.Sprintf("/api/verticals/%s", vertical.Slug))
					if err != nil {
						return err
					}
					query := url.Values{}
					query.Add("service", u.Service)
					query.Add("scopeService", u.ScopeService)
					path.RawQuery = query.Encode()
					if _, _, err := keystone.PutJSON(client, headers, path, av); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// DownloadPanel reads a single panel
func (u *Urbo) DownloadPanel(client keystone.HTTPClient, headers http.Header, slug string) (json.RawMessage, error) {
	path, err := u.URL.Parse(fmt.Sprintf("/api/panels/%s", slug))
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	// parameters learnt by watching urbo, not actually documented...
	// make sure the panels are returned in a form we can copy and paste them in urbo.
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

// DownloadPanel reads a single panel
func (u *Urbo) UploadPanel(client keystone.HTTPClient, headers http.Header, panel json.RawMessage) error {
	var buffer map[string]interface{}
	if err := json.Unmarshal(panel, &buffer); err != nil {
		return err
	}
	slug, ok := buffer["slug"]
	if !ok {
		return errors.New("panel does not have slug")
	}
	textSlug, ok := slug.(string)
	if !ok {
		return fmt.Errorf("slug %v is not a string", slug)
	}
	buffer["service"] = u.Service
	validPanel, err := json.Marshal(buffer)
	if err != nil {
		return err
	}
	path, err := u.URL.Parse(fmt.Sprintf("/api/panels"))
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Add("service", u.Service)
	query.Add("scopeService", u.ScopeService)
	path.RawQuery = query.Encode()
	if _, _, err := keystone.PostJSON(client, headers, path, validPanel); err != nil {
		var netErr keystone.NetError
		if errors.As(err, &netErr) {
			if netErr.StatusCode == 400 { // Panel already exists
				path, err := u.URL.Parse(fmt.Sprintf("/api/panels/%s:%s", u.Service, textSlug))
				if err != nil {
					return err
				}
				query := url.Values{}
				query.Add("service", u.Service)
				query.Add("scopeService", u.ScopeService)
				path.RawQuery = query.Encode()
				if _, _, err := keystone.PutJSON(client, headers, path, validPanel); err != nil {
					return err
				}
			}
		}
	}
	return nil
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
