package keystone

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		io.Copy(io.Discard, resp.Body)
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
			payload, newErr = io.ReadAll(resp.Body)
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

// Just enough model of the auth response to get to the user id
type authReply struct {
	Token struct {
		User struct {
			Id string `json:"id"`
		} `json:"user"`
	} `json:"token"`
}

// Login into the Context Broker, get a session token
func (o *Keystone) Login(client HTTPClient, password string, retries Backoff) (string, string, error) {
	payload := fmt.Sprintf(
		`{"auth": {"identity": {"methods": ["password"], "password": {"user": {"domain": {"name": %q}, "name": %q, "password": %q}}}, "scope": {"domain": {"name": %q}}}}`,
		o.Service, o.Username, password, o.Service,
	)
	loginURL, err := o.URL.Parse("/v3/auth/tokens")
	if err != nil {
		return "", "", err
	}
	var current int
	for {
		header, body, err := PostJSON(client, nil, loginURL, payload)
		if err == nil {
			var (
				reply  authReply
				userId string
			)
			if err := json.Unmarshal(body, &reply); err != nil {
				log.Printf("Failed to parse auth reply, will not propagate user id: %s", err)
			} else {
				userId = reply.Token.User.Id
			}
			return header.Get("X-Subject-Token"), userId, nil
		}
		// retry errors 500
		var netErr NetError
		if errors.As(err, &netErr) {
			if netErr.StatusCode != 500 {
				return "", "", err
			}
		}
		retry, delay := retries.KeepTrying(current)
		current += 1
		if !retry {
			return "", "", err
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
func GetPaginatedJSON(client HTTPClient, headers http.Header, path *url.URL, p Paginator, allowUnknownFields bool, maximum int) error {
	offset, limit, total := 0, 50, 50
	for offset < total {
		if total > 2*limit {
			// If it's going to tske long, then print a progress indicator
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
			err := p.Append(raw, allowUnknownFields)
			if err != nil {
				return err
			}
		}
		offset += len(data)
		if maximum > 0 && total > maximum {
			total = maximum
		}
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
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maximumPayload))
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
		bodyBytes, err := io.ReadAll(resp.Body)
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

type keystoneDomains struct {
	Links   json.RawMessage `json:"links,omitempty"`
	Domains []fiware.Domain `json:"domains"`
}

func (k *Keystone) Domains(client HTTPClient, headers http.Header, enabled bool) ([]fiware.Domain, error) {
	urlProjects, err := k.URL.Parse("/v3/domains")
	if err != nil {
		return nil, err
	}
	if !enabled {
		urlProjects.Query().Add("enabled", "false")
	}
	var projects keystoneDomains
	if _, err := Query(client, http.MethodGet, headers, urlProjects, &projects, true); err != nil {
		return nil, err
	}
	return projects.Domains, nil
}

type domainList struct {
	Links   json.RawMessage `json:"links,omitempty"`
	Domains []domainInfo    `json:"domains"`
}

type domainInfo struct {
	Links       json.RawMessage `json:"links,omitempty"`
	Description string          `json:"description"`
	Tags        json.RawMessage `json:"tags,omitempty"`
	Enabled     bool            `json:"enabled"`
	ID          string          `json:"id"`
	Name        string          `json:"name"`
}

func (k *Keystone) myDomain(client HTTPClient, headers http.Header) (string, string, error) {
	// Get the domain id for the service
	urlDomain, err := k.URL.Parse("/v3/auth/domains")
	if err != nil {
		return "", "", err
	}
	var domain domainList
	if err := GetJSON(client, headers, urlDomain, &domain, true); err != nil {
		return "", "", err
	}
	if len(domain.Domains) == 0 {
		return "", "", errors.New("no domains found")
	}
	domName := domain.Domains[0].Name
	domID := domain.Domains[0].ID
	return domName, domID, nil
}

type keystoneUsers struct {
	Links json.RawMessage `json:"links,omitempty"`
	Users []fiware.User   `json:"users"`
}

func (k *Keystone) Users(client HTTPClient, headers http.Header) ([]fiware.User, error) {
	// Get the domain id for the service
	_, domID, err := k.myDomain(client, headers)
	if err != nil {
		return nil, err
	}
	urlProjects, err := k.URL.Parse(fmt.Sprintf("/v3/users?domain_id=%s", domID))
	if err != nil {
		return nil, err
	}
	var users keystoneUsers
	if _, err := Query(client, http.MethodGet, headers, urlProjects, &users, false); err != nil {
		return nil, err
	}
	return users.Users, nil
}

type keystoneGroups struct {
	Links  json.RawMessage `json:"links,omitempty"`
	Groups []fiware.Group  `json:"groups"`
}

func (k *Keystone) Groups(client HTTPClient, headers http.Header) ([]fiware.Group, error) {
	// Get the domain id for the service
	_, domID, err := k.myDomain(client, headers)
	if err != nil {
		return nil, err
	}
	urlGroups, err := k.URL.Parse(fmt.Sprintf("/v3/groups?domain_id=%s", domID))
	if err != nil {
		return nil, err
	}
	var groups keystoneGroups
	if _, err := Query(client, http.MethodGet, headers, urlGroups, &groups, false); err != nil {
		return nil, err
	}
	for idx, grp := range groups.Groups {
		urlUsers, err := k.URL.Parse(fmt.Sprintf("/v3/groups/%s/users", grp.ID))
		if err != nil {
			return nil, err
		}
		var users keystoneUsers
		if _, err := Query(client, http.MethodGet, headers, urlUsers, &users, false); err != nil {
			return nil, err
		}
		log.Printf("Group %s has %d users", grp.Name, len(users.Users))
		usrList := make([]string, 0, len(users.Users))
		for _, user := range users.Users {
			usrList = append(usrList, user.ID)
		}
		groups.Groups[idx].Users = usrList
	}
	return groups.Groups, nil
}

type keystoneRoles struct {
	Links json.RawMessage `json:"links,omitempty"`
	Roles []fiware.Role   `json:"roles"`
}

func (k *Keystone) Roles(client HTTPClient, headers http.Header) ([]fiware.Role, error) {
	// Get the domain id for the service
	// domName, domID, err := k.myDomain(client, headers)
	// if err != nil {
	// 	return nil, err
	// }
	urlRoles, err := k.URL.Parse("/v3/roles")
	if err != nil {
		return nil, err
	}
	var roles keystoneRoles
	if _, err := Query(client, http.MethodGet, headers, urlRoles, &roles, false); err != nil {
		return nil, err
	}
	// for idx := range roles.Roles {
	// 	roles.Roles[idx].Domain = domName
	// }
	return roles.Roles, nil
}

type keystoneRoleAssignments struct {
	Links       json.RawMessage         `json:"links,omitempty"`
	Assignments []fiware.RoleAssignment `json:"role_assignments"`
}

func (k *Keystone) UserRoles(client HTTPClient, headers http.Header, uid string) ([]fiware.RoleAssignment, error) {
	return k.assignments(client, headers, "user.id", uid)
}

func (k *Keystone) GroupRoles(client HTTPClient, headers http.Header, gid string) ([]fiware.RoleAssignment, error) {
	return k.assignments(client, headers, "group.id", gid)
}

func (k *Keystone) assignments(client HTTPClient, headers http.Header, param, val string) ([]fiware.RoleAssignment, error) {
	urlAssignments, err := k.URL.Parse(fmt.Sprintf("/v3/role_assignments?%s=%s", param, val))
	if err != nil {
		return nil, err
	}
	var assignments keystoneRoleAssignments
	if _, err := Query(client, http.MethodGet, headers, urlAssignments, &assignments, false); err != nil {
		return nil, err
	}
	// Tomo nota de todos los roles heredados
	inherit := make(map[string]string)
	scopes := make([]fiware.AssignmentScope, 0, len(assignments.Assignments))
	for _, assign := range assignments.Assignments {
		scope, err := assign.ParseScope()
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
		if scope.Domain != "" && scope.Inherited != "" && scope.Inherited == "projects" {
			inherit[assign.Role.ID] = scope.Inherited
		}
	}
	// Y elimino del resultado las asignaciones redundantes
	result := make([]fiware.RoleAssignment, 0, len(assignments.Assignments))
	for idx, assign := range assignments.Assignments {
		skip_assign := false
		scope := scopes[idx]
		if scope.Project != "" && assign.Role.ID != "" {
			_, found := inherit[assign.Role.ID]
			if found {
				skip_assign = true
			}
		}
		if !skip_assign {
			result = append(result, assign)
		}
	}
	return result, nil
}
