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
		// HACK: No quiero que se añada a las acciones el servicio y el subservicio,
		// a menos que use variables (contenga "${}").
		// Así que decodifico la acción y compruebo si tiene los campos
		// "service" y "subservice", y de ser así los omito.
		var actionList []map[string]interface{}
		if len(rule.Action) > 0 {
			var action interface{}
			if err := json.Unmarshal(rule.Action, &action); err == nil {
				actionList = make([]map[string]interface{}, 0, 4)
				allMaps := true
				switch action := action.(type) {
				case map[string]interface{}:
					actionList = append(actionList, action)
				case []interface{}:
					for _, item := range action {
						m, ok := item.(map[string]interface{})
						if !ok {
							allMaps = false
							break
						}
						actionList = append(actionList, m)
					}
				default:
					allMaps = false
				}
				// Si no he podido procesarlo todo,
				// vaciar la lista
				if !allMaps {
					actionList = nil
				}
			}
		}
		var replaced bool // indica si hemos reemplazado alguna accion
		if len(actionList) > 0 {
			// PAra prever el caso en que descargamos reglas de
			// un servicio, y queremos aplicarlas a otro.
			keys := map[string]string{
				"service":    rule.Service,
				"subservice": rule.Subservice,
			}
			for index, current := range actionList {
				// Preservamos todas las claves si al menos una es variable
				preserve := false
				for key, defValue := range keys {
					if k, ok := current[key]; ok {
						if s, ok := k.(string); ok {
							if s != defValue {
								preserve = true
							}
						}
					}
				}
				if !preserve {
					for key := range keys {
						if _, ok := current[key]; ok {
							delete(current, key)
							replaced = true
							fmt.Printf("Removing attribute %s from action %d in rule %s\n", key, index, rule.Name)
						}
					}
					actionList[index] = current
				}
			}
		}
		if replaced {
			var newAction interface{} = actionList
			if len(actionList) == 1 {
				newAction = actionList[0]
			}
			if newBytes, err := json.Marshal(newAction); err == nil {
				rule.Action = newBytes
			}
		}
		// FIN DE HACK
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
		if _, _, err := keystone.Update(client, http.MethodPost, headers, path, rule); err != nil {
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
