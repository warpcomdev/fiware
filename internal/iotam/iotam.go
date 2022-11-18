package iotam

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-multierror"

	"github.com/warpcomdev/fiware"
	"github.com/warpcomdev/fiware/internal/keystone"
)

type Iotam struct {
	URL                *url.URL
	AllowUnknownFields bool
}

// New IoTA API manager
func New(iotaURL string) (*Iotam, error) {
	apiURL, err := url.Parse(iotaURL)
	if err != nil {
		return nil, err
	}
	return &Iotam{
		URL: apiURL,
	}, nil
}

// Services reads the list of groups from the IoTA Manager
func (i *Iotam) Services(client keystone.HTTPClient, headers http.Header) ([]fiware.Service, error) {
	path, err := i.URL.Parse("iot/services")
	if err != nil {
		return nil, err
	}
	var response struct {
		Count    int              `json:"count"`
		Services []fiware.Service `json:"services"`
	}
	if err := keystone.GetJSON(client, headers, path, &response, i.AllowUnknownFields); err != nil {
		return nil, err
	}
	return response.Services, nil
}

// Devices reads the list of devices from the IoTA Manager
func (i *Iotam) Devices(client keystone.HTTPClient, headers http.Header) ([]fiware.Device, error) {
	path, err := i.URL.Parse("iot/devices")
	if err != nil {
		return nil, err
	}
	var response struct {
		Count   int             `json:"count"`
		Devices []fiware.Device `json:"devices"`
	}
	if err := keystone.GetJSON(client, headers, path, &response, i.AllowUnknownFields); err != nil {
		return nil, err
	}
	return response.Devices, nil
}

// PostServices sends a POST request for a set of Services
func (i *Iotam) PostServices(client keystone.HTTPClient, headers http.Header, services []fiware.Service) error {
	clean := make([]fiware.Service, 0, len(services))
	for _, service := range services {
		service.GroupStatus = fiware.GroupStatus{}
		clean = append(clean, service)
	}
	// Aggregate Devices by protocol
	resourceMap, err := groupResources(services, func(g fiware.Service) string { return g.Protocol })
	if err != nil {
		return err
	}
	for protocol, services := range resourceMap {
		path, err := i.protocolURL("iot/services", protocol)
		if err != nil {
			return err
		}
		request := struct {
			Services []fiware.Service `json:"services"`
		}{Services: services}
		if _, _, err := keystone.PostJSON(client, headers, path, request); err != nil {
			return err
		}
	}
	return nil
}

// DeleteServices sends a DELETE request for a set of Services
func (i *Iotam) DeleteServices(client keystone.HTTPClient, headers http.Header, services []fiware.Service) error {
	var errList error
	for _, service := range services {
		if service.Resource == "" || service.APIKey == "" || service.Protocol == "" {
			return errors.New("All devices must have protocol, resource and apikey")
		}
		path, err := i.URL.Parse("iot/services")
		if err != nil {
			return err
		}
		query := path.Query()
		query.Add("resource", service.Resource)
		query.Add("apikey", service.APIKey)
		query.Add("protocol", service.Protocol)
		path.RawQuery = query.Encode()
		if _, err := keystone.Query(client, http.MethodDelete, headers, path, nil, true); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}

// PostDevices sends a POST request for a set of Devices
func (i *Iotam) PostDevices(client keystone.HTTPClient, headers http.Header, devices []fiware.Device) error {
	clean := make([]fiware.Device, 0, len(devices))
	for _, service := range devices {
		service.DeviceStatus = fiware.DeviceStatus{}
		clean = append(clean, service)
	}
	resourceMap, err := groupResources(clean, func(g fiware.Device) string { return g.Protocol })
	if err != nil {
		return err
	}
	for protocol, devices := range resourceMap {
		path, err := i.protocolURL("iot/devices", protocol)
		if err != nil {
			return err
		}
		request := struct {
			Devices []fiware.Device `json:"devices"`
		}{Devices: devices}
		if _, _, err := keystone.PostJSON(client, headers, path, request); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDevices sends a DELETE request for a set of Devices
func (i *Iotam) DeleteDevices(client keystone.HTTPClient, headers http.Header, devices []fiware.Device) error {
	var errList error
	for _, device := range devices {
		if device.DeviceId == "" {
			return errors.New("All devices must have a deviceId")
		}
		path, err := i.protocolURL(fmt.Sprintf("iot/devices/%s", device.DeviceId), device.Protocol)
		if err != nil {
			return err
		}
		if _, err := keystone.Query(client, http.MethodDelete, headers, path, nil, true); err != nil {
			errList = multierror.Append(errList, err)
		}
	}
	return errList
}

func groupResources[R any](resources []R, indexFunc func(R) string) (map[string][]R, error) {
	resourceMap := make(map[string][]R)
	for _, res := range resources {
		protocol := indexFunc(res)
		if protocol == "" {
			return nil, errors.New("all resources must have a `protocol` field")
		}
		group := append(resourceMap[protocol], res)
		resourceMap[protocol] = group
	}
	return resourceMap, nil
}

func (i *Iotam) protocolURL(path, protocol string) (*url.URL, error) {
	newURL, err := i.URL.Parse(path)
	if err != nil {
		return nil, err
	}
	query := newURL.Query()
	query.Add("protocol", protocol)
	newURL.RawQuery = query.Encode()
	return newURL, nil
}
