package perseo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-multierror"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/keystone"
)

// PErseo manages connections to Perseo API
type Perseo struct {
	URL                *url.URL
	AllowUnknownFields bool
}

// New Perseo API instance
func New(perseoURL string) (*Perseo, error) {
	apiURL, err := url.Parse(perseoURL)
	if err != nil {
		return nil, err
	}
	return &Perseo{
		URL: apiURL,
	}, nil
}

// Rules reads the list of rules from Perseo
func (o *Perseo) Rules(client *http.Client, headers http.Header) ([]fiware.Rule, error) {
	path, err := o.URL.Parse("rules")
	if err != nil {
		return nil, err
	}
	var response struct {
		Count int             `json:"count"`
		Data  []fiware.Rule   `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	if err := keystone.GetJSON(client, headers, path, &response, o.AllowUnknownFields); err != nil {
		return nil, err
	}
	if len(response.Error) > 0 && !bytes.Equal(response.Error, []byte("null")) {
		return nil, fmt.Errorf("perseo replied with error: %s", string(response.Error))
	}
	return response.Data, nil
}

// PostRules posts a list of rules to Perseo
func (o *Perseo) PostRules(client *http.Client, headers http.Header, rules []fiware.Rule) error {
	var errList error
	for _, rule := range rules {
		rule.RuleStatus = fiware.RuleStatus{}
		if rule.Name == "" {
			return errors.New("All rules must have name")
		}
		if rule.Text != "" {
			rule.NoSignal = nil // those two are mutually exclusive
		}
		path, err := o.URL.Parse("rules")
		if err != nil {
			return err
		}
		if _, err := keystone.Update(client, http.MethodPost, headers, path, rule); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}

// DeleteRules deletes a list of rules from Perseo
func (o *Perseo) DeleteRules(client *http.Client, headers http.Header, rules []fiware.Rule) error {
	var errList error
	for _, rule := range rules {
		rule.RuleStatus = fiware.RuleStatus{}
		if rule.Name == "" {
			return errors.New("All rules must have name")
		}
		path, err := o.URL.Parse(fmt.Sprintf("rules/%s", rule.Name))
		if err != nil {
			return err
		}
		if err := keystone.Query(client, http.MethodDelete, headers, path, nil, false); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}
