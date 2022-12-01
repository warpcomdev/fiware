package perseo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/hashicorp/go-multierror"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/keystone"
)

// Perseo manages connections to Perseo API
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

// rulenameRegext matches rule name section
var rulenameRegexp *regexp.Regexp = regexp.MustCompile(`(?i)select\s+[^\s]+\s+as ruleName,\s*`)

// Rules reads the list of rules from Perseo
func (o *Perseo) Rules(client keystone.HTTPClient, headers http.Header) ([]fiware.Rule, error) {
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
	// HACK: no voy a volcar el ruleName, voy a dejar que lo ponga perseo
	for index, data := range response.Data {
		if data.Text != "" {
			data.Text = rulenameRegexp.ReplaceAllLiteralString(data.Text, "select ")
			response.Data[index] = data
		}
	}
	// FIN DE HACK
	// HACK: Me aseguro de que todas las acciones son listas
	for index, rule := range response.Data {
		if rule.Action != nil && len(rule.Action) > 0 {
			reader := bytes.NewReader(rule.Action)
			if rune, size, err := reader.ReadRune(); err != nil && size > 0 && rune != '[' {
				var newAction bytes.Buffer
				newAction.WriteString("[")
				newAction.Write(rule.Action)
				newAction.WriteString("]")
				response.Data[index].Action = json.RawMessage(newAction.Bytes())
			}
		}
	}
	// FIN DE HACK
	return response.Data, nil
}

// PostRules posts a list of rules to Perseo
func (o *Perseo) PostRules(client keystone.HTTPClient, headers http.Header, rules []fiware.Rule) error {
	var errList error
	for _, rule := range rules {
		// HACK: No quiero que se añada a las acciones el servicio y el subservicio,
		// a menos que use variables (contenga "${}").
		// Así que decodifico la acción y compruebo si tiene los campos
		// "service" y "subservice", y de ser así los omito.
		actionList := rule.ActionList()
		replaced := false // indica si hemos reemplazado alguna accion
		if actionList != nil && len(actionList) > 0 {
			// Para prever el caso en que descargamos reglas de
			// un servicio, y queremos aplicarlas a otro.
			keys := map[string]string{
				"service":    rule.Service,
				"subservice": rule.Subservice,
			}
			for index, current := range actionList {
				switch current := current.(type) {
				case map[string]interface{}:
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
		}
		if replaced {
			if newBytes, err := json.Marshal(actionList); err == nil {
				rule.Action = newBytes
			}
		}
		// FIN DE HACK
		rule.RuleStatus = fiware.RuleStatus{}
		if rule.Name == "" {
			return errors.New("All rules must have name")
		}
		if rule.Text != "" {
			// HACK 2: no voy a subir el ruleName, voy a dejar que lo ponga perseo
			rule.Text = rulenameRegexp.ReplaceAllLiteralString(rule.Text, "select ")
			// FIN DE HACK 2
			if rule.NoSignal != nil && len(rule.NoSignal) > 0 {
				return fmt.Errorf("both rule.Text and rule.NoSignal defined for rule %s", rule.Name)
			}
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
func (o *Perseo) DeleteRules(client keystone.HTTPClient, headers http.Header, rules []fiware.Rule) error {
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
		if _, err := keystone.Query(client, http.MethodDelete, headers, path, nil, false); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}
