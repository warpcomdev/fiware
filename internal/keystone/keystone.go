package keystone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/warpcomdev/fiware"
)

// Keystone manages Requests to the Identity Manager
type Keystone struct {
	URL               *url.URL
	Username, Service string
}

// New Keystone client instance
func New(keystoneURL string, username, service string) (*Keystone, error) {
	URL, err := url.Parse(fmt.Sprintf("%s", keystoneURL))
	if err != nil {
		return nil, err
	}
	return &Keystone{
		URL:      URL,
		Username: username,
		Service:  service,
	}, nil
}

// Exhaust reads the response body until completion, and closes it.
func Exhaust(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

// NetError describes an error performing a request
type NetError struct {
	Req        http.Request
	StatusCode int
	Resp       []byte
	Err        error
}

// Error implements error
func (n NetError) Error() string {
	base := fmt.Sprintf("%s request to %s failed with code %d", n.Req.Method, n.Req.URL.String(), n.StatusCode)
	if n.Err != nil {
		return fmt.Sprintf("%s, and body could not be read: %v", base, n.Err)
	}
	if n.Resp != nil {
		return fmt.Sprintf("%s: %s", base, string(n.Resp))
	}
	return base
}

// Unwrap implements errors.Unwrap
func (n NetError) Unwrap() error {
	return n.Err
}

// newNetError builds an error from a Request and unexpected Response
func newNetError(req *http.Request, resp *http.Response) error {
	// Do not propagate body or headers of the request, might contain
	// creedentials or other sensitive data
	anonymousReq := http.Request{
		URL:           req.URL,
		Method:        req.Method,
		ContentLength: req.ContentLength,
	}
	var (
		payload    []byte
		err        error
		statusCode int
	)
	if resp != nil {
		statusCode = resp.StatusCode
		if resp.Body != nil {
			payload, err = ioutil.ReadAll(resp.Body)
		}
	}
	return NetError{
		Req:        anonymousReq,
		StatusCode: statusCode,
		Resp:       payload,
		Err:        err,
	}
}

// Login into the Context Broker, get a session token
func (o *Keystone) Login(client *http.Client, password string) (string, error) {
	payload := fmt.Sprintf(
		`{"auth": {"identity": {"methods": ["password"], "password": {"user": {"domain": {"name": %q}, "name": %q, "password": %q}}}, "scope": {"domain": {"name": %q}}}}`,
		o.Service, o.Username, password, o.Service,
	)
	loginURL, err := o.URL.Parse("/v3/auth/tokens")
	if err != nil {
		return "", err
	}
	header, _, err := PostJSON(client, nil, loginURL, payload)
	if err != nil {
		return "", err
	}
	return header.Get("X-Subject-Token"), nil
}

// Headers returns the authentication headers for a subservice
func (o *Keystone) Headers(subservice, token string) http.Header {
	h := make(http.Header)
	if !strings.HasPrefix(subservice, "/") {
		subservice = "/" + subservice
	}
	h.Add("Fiware-Service", o.Service)
	h.Add("Fiware-ServicePath", subservice)
	h.Add("X-Auth-Token", token)
	return h
}

// DecodeError returned when failed to decode json data
type DecodeError struct {
	Data json.RawMessage
	Err  error
}

// Error implements error
func (d DecodeError) Error() string {
	return fmt.Sprintf("failed to parse '%s': %v", string(d.Data), d.Err)
}

// Unwrap implements errors.Unwrap
func (d DecodeError) Unwrap() error {
	return d.Err
}

const maximumPayload = 16 * 1024 * 1024 // 16MB should be large enough

// GetJSON is a convenience wrapper for Query(client, http.MethodGet, ...)
// TODO: Add a variant with pagination support
func GetJSON(client *http.Client, headers http.Header, path *url.URL, data interface{}, allowUnknownFields bool) error {
	return Query(client, http.MethodGet, headers, path, data, allowUnknownFields)
}

// PostJSON is a convenience wrapper for Update(client, http.MethodPost, ...)
func PostJSON(client *http.Client, headers http.Header, path *url.URL, data interface{}) (http.Header, []byte, error) {
	return Update(client, http.MethodPost, headers, path, data)
}

// Query performs an HTTP request without payload, loads the result into `data`
func Query(client *http.Client, method string, headers http.Header, path *url.URL, data interface{}, allowUnknownFields bool) error {

	req := &http.Request{
		Header: headers,
		URL:    path,
		Method: method,
	}
	resp, err := client.Do(req)
	defer Exhaust(resp)
	if err != nil {
		return newNetError(req, nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newNetError(req, resp)
	}
	if data == nil { // payload not required
		return nil
	}
	raw, err := ioutil.ReadAll(io.LimitReader(resp.Body, maximumPayload))
	if err != nil {
		return newNetError(req, resp)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if !allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(data); err != nil {
		return DecodeError{
			Data: raw,
			Err:  err,
		}
	}
	return nil
}

type pager interface {
	Next() interface{} // Return a buffer for next page
	Done() int         // Must return number of items received in interface{}
}

// Query performs an HTTP request without payload, loads the result into `data`
func page(client *http.Client, method string, headers http.Header, path *url.URL, data pager, allowUnknownFields bool) error {
	q := path.Query()
	limit := 50
	offset := 0
	top := 1000 // this is our hard limit
	for offset < top {
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		path.RawQuery = q.Encode()
		if err := Query(client, method, headers, path, data.Next(), allowUnknownFields); err != nil {
			return err
		}
		if recv := data.Done(); recv < limit {
			return nil
		}
		<-time.After(time.Second) // add some delay to avoid overwhelming the CB
		offset = offset + limit
	}
	return nil
}

// Update performs an HTTP request with JSON payload, returns headers.
func Update(client *http.Client, method string, headers http.Header, path *url.URL, data interface{}) (http.Header, []byte, error) {

	// Serialize request to bytes
	var dataBytes []byte
	switch data := data.(type) {
	case string:
		dataBytes = []byte(data)
	case []byte:
		dataBytes = data
	default:
		var err error
		if dataBytes, err = json.Marshal(data); err != nil {
			return nil, nil, err
		}
	}

	// Clone headers and add content type
	var newHeaders http.Header
	if headers == nil {
		newHeaders = make(http.Header)
	} else {
		newHeaders = headers.Clone()
	}
	newHeaders.Add("Content-Type", "application/json")

	// Perform Request
	req := &http.Request{
		Header:        newHeaders,
		URL:           path,
		Method:        method,
		Body:          io.NopCloser(bytes.NewReader(dataBytes)),
		ContentLength: int64(len(dataBytes)),
	}
	resp, err := client.Do(req)
	defer Exhaust(resp)

	// Manage response
	if err != nil {
		return nil, nil, newNetError(req, nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, newNetError(req, resp)
	}
	if resp.StatusCode != 204 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, newNetError(req, resp)
		}
		return resp.Header, bodyBytes, nil
	}
	return resp.Header, nil, nil
}

type keystoneProjects struct {
	Links    json.RawMessage  `json:"links,omitempty"`
	Projects []fiware.Project `json:"projects"`
}

func (k *Keystone) Projects(client *http.Client, headers http.Header) ([]fiware.Project, error) {
	urlProjects, err := k.URL.Parse("/v3/auth/projects")
	if err != nil {
		return nil, err
	}
	var projects keystoneProjects
	if err := Query(client, http.MethodGet, headers, urlProjects, &projects, true); err != nil {
		return nil, err
	}
	return projects.Projects, nil
}
