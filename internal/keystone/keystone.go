package keystone

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/warpcomdev/fiware"
)

// HTTPClient encapsulates the funcionality required from *http.Client.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

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
	Req         http.Request
	StatusCode  int
	RespHeaders http.Header
	Resp        []byte
	Err         error
}

// Error implements error
func (n NetError) Error() string {
	base := strings.Builder{}
	fmt.Fprintf(&base, "%s request to %s failed with code %d\n", n.Req.Method, n.Req.URL.String(), n.StatusCode)
	switch {
	case n.Err != nil:
		fmt.Fprintf(&base, "body could not be read: %v", n.Err)
	case n.Resp != nil:
		n.RespHeaders.Write(&base)
		base.WriteString("\n")
		base.WriteString(string(n.Resp))
	}
	return base.String()
}

// Unwrap implements errors.Unwrap
func (n NetError) Unwrap() error {
	return n.Err
}

// newNetError builds an error from a Request and unexpected Response
func newNetError(req *http.Request, resp *http.Response, err error) error {
	// Do not propagate body or headers of the request, might contain
	// creedentials or other sensitive data
	anonymousReq := http.Request{
		URL:           req.URL,
		Method:        req.Method,
		ContentLength: req.ContentLength,
	}
	var (
		payload    []byte
		statusCode int
		headers    http.Header
	)
	if resp != nil {
		statusCode = resp.StatusCode
		headers = resp.Header
		if resp.Body != nil {
			// Only override err if nil
			var newErr error
			payload, newErr = ioutil.ReadAll(resp.Body)
			if newErr != nil {
				err = newErr
			}
		}
	}
	return NetError{
		Req:         anonymousReq,
		StatusCode:  statusCode,
		RespHeaders: headers,
		Resp:        payload,
		Err:         err,
	}
}

// Backoff controls retry policy
type Backoff interface {
	KeepTrying(retries int) (bool, time.Duration)
}

// LinealBackoff performs lineal backoff
type LinealBackoff struct {
	MaxRetries int
	Delay      time.Duration
}

// KeepTrying implements Retry
func (l LinealBackoff) KeepTrying(retries int) (bool, time.Duration) {
	return (retries < l.MaxRetries), l.Delay
}

// ExponentialBackoff performs exponential backoff
type ExponentialBackoff struct {
	MaxRetries   int
	InitialDelay time.Duration
	DelayFactor  float64
	MaxDelay     time.Duration
}

// KeepTrying implements Retry
func (l ExponentialBackoff) KeepTrying(retries int) (bool, time.Duration) {
	targetDelay := time.Duration(float64(l.InitialDelay) * math.Pow(l.DelayFactor, float64(retries)))
	if targetDelay > l.MaxDelay {
		targetDelay = l.MaxDelay
	}
	return (retries < l.MaxRetries), targetDelay
}

// Login into the Context Broker, get a session token
func (o *Keystone) Login(client HTTPClient, password string, retries Backoff) (string, error) {
	payload := fmt.Sprintf(
		`{"auth": {"identity": {"methods": ["password"], "password": {"user": {"domain": {"name": %q}, "name": %q, "password": %q}}}, "scope": {"domain": {"name": %q}}}}`,
		o.Service, o.Username, password, o.Service,
	)
	loginURL, err := o.URL.Parse("/v3/auth/tokens")
	if err != nil {
		return "", err
	}
	var current int
	for {
		header, _, err := PostJSON(client, nil, loginURL, payload)
		if err == nil {
			return header.Get("X-Subject-Token"), nil
		}
		// retry errors 500
		var netErr NetError
		if errors.As(err, &netErr) {
			if netErr.StatusCode != 500 {
				return "", err
			}
		}
		retry, delay := retries.KeepTrying(current)
		current += 1
		if !retry {
			return "", err
		}
		<-time.After(delay)
	}
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
	Type interface{}
	Data json.RawMessage
	Err  error
}

// Error implements error
func (d DecodeError) Error() string {
	return fmt.Sprintf("failed to parse '%s' into '%s': %v", string(d.Data), fmt.Sprintf("%T", d.Type), d.Err)
}

// Unwrap implements errors.Unwrap
func (d DecodeError) Unwrap() error {
	return d.Err
}

const maximumPayload = 16 * 1024 * 1024 // 16MB should be large enough

// GetJSON is a convenience wrapper for Query(client, http.MethodGet, ...)
// TODO: Add a variant with pagination support
func GetJSON(client HTTPClient, headers http.Header, path *url.URL, data interface{}, allowUnknownFields bool) error {
	_, err := Query(client, http.MethodGet, headers, path, data, allowUnknownFields)
	return err
}

type Paginator interface {
	Append(item json.RawMessage, allowUnknownFields bool) error
}

// SlicePaginator is a generic type of Paginator based on a slice
type SlicePaginator[T any] struct {
	Slice []T
}

// Append implements Paginator
func (s *SlicePaginator[T]) Append(raw json.RawMessage, allowUnknownFields bool) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if !allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	var subs T
	if err := decoder.Decode(&subs); err != nil {
		return fmt.Errorf("Failed to decode %T from %s: %w", s.Slice, string(raw), err)
	}
	s.Slice = append(s.Slice, subs)
	return nil
}

// NewPaginator creates a new paginator backed by the given slice
func NewPaginator[T any](slice []T) *SlicePaginator[T] {
	return &SlicePaginator[T]{
		Slice: slice,
	}
}

// GetPaginatedJSON is a convenience wrapper for Query(client, http.MethodGet, ...)
func GetPaginatedJSON(client HTTPClient, headers http.Header, path *url.URL, p Paginator, allowUnknownFields bool) error {
	offset, limit, total := 0, 50, 50
	for offset < total {
		if total > 2*limit {
			// If it's going to tke long, then print a progress indicator
			log.Printf("Getting %d items of %d at offset %d", limit, total, offset)
		}
		limitedURL := *path // make a copy
		values := limitedURL.Query()
		remain := total - offset
		if remain > limit {
			remain = limit
		}
		values.Add("offset", strconv.Itoa(offset))
		values.Add("limit", strconv.Itoa(remain))
		values.Add("options", "count")
		limitedURL.RawQuery = values.Encode()
		var data []json.RawMessage
		header, err := Query(client, http.MethodGet, headers, &limitedURL, &data, allowUnknownFields)
		if err != nil {
			return err
		}
		total, err = strconv.Atoi(header.Get("Fiware-Total-Count"))
		if err != nil {
			return err
		}
		for _, raw := range data {
			if err := p.Append(raw, allowUnknownFields); err != nil {
				return err
			}
		}
		offset += len(data)
	}
	return nil
}

// PostJSON is a convenience wrapper for Update(client, http.MethodPost, ...)
func PostJSON(client HTTPClient, headers http.Header, path *url.URL, data interface{}) (http.Header, []byte, error) {
	return Update(client, http.MethodPost, headers, path, data)
}

// PutJSON is a convenience wrapper for Update(client, http.MethodPut, ...)
func PutJSON(client HTTPClient, headers http.Header, path *url.URL, data interface{}) (http.Header, []byte, error) {
	return Update(client, http.MethodPut, headers, path, data)
}

// Query performs an HTTP request without payload, loads the result into `data`
func Query(client HTTPClient, method string, headers http.Header, path *url.URL, data interface{}, allowUnknownFields bool) (http.Header, error) {

	req := &http.Request{
		Header: headers,
		URL:    path,
		Method: method,
	}
	resp, err := client.Do(req)
	defer Exhaust(resp)
	if err != nil {
		return nil, newNetError(req, nil, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newNetError(req, resp, nil)
	}
	if data == nil { // payload not required
		return resp.Header, nil
	}
	raw, err := ioutil.ReadAll(io.LimitReader(resp.Body, maximumPayload))
	if err != nil {
		return nil, newNetError(req, resp, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if !allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(data); err != nil {
		return nil, DecodeError{
			Type: data,
			Data: raw,
			Err:  err,
		}
	}
	return resp.Header, nil
}

type pager interface {
	Next() interface{} // Return a buffer for next page
	Done() int         // Must return number of items received in interface{}
}

// Query performs an HTTP request without payload, loads the result into `data`
func page(client HTTPClient, method string, headers http.Header, path *url.URL, data pager, allowUnknownFields bool) error {
	q := path.Query()
	limit := 50
	offset := 0
	top := 1000 // this is our hard limit
	for offset < top {
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		path.RawQuery = q.Encode()
		if _, err := Query(client, method, headers, path, data.Next(), allowUnknownFields); err != nil {
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
func Update(client HTTPClient, method string, headers http.Header, path *url.URL, data interface{}) (http.Header, []byte, error) {

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
		return nil, nil, newNetError(req, nil, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, newNetError(req, resp, nil)
	}
	if resp.StatusCode != 204 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, newNetError(req, resp, err)
		}
		return resp.Header, bodyBytes, nil
	}
	return resp.Header, nil, nil
}

type keystoneProjects struct {
	Links    json.RawMessage  `json:"links,omitempty"`
	Projects []fiware.Project `json:"projects"`
}

func (k *Keystone) Projects(client HTTPClient, headers http.Header) ([]fiware.Project, error) {
	urlProjects, err := k.URL.Parse("/v3/auth/projects")
	if err != nil {
		return nil, err
	}
	var projects keystoneProjects
	if _, err := Query(client, http.MethodGet, headers, urlProjects, &projects, true); err != nil {
		return nil, err
	}
	return projects.Projects, nil
}
