//go:build !windows

package main

import (
	"crypto/tls"
	"net/http"
	"time"
)

func _httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// TODO: Hacer esto configurable
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// Para entornos con VPNS restrictivas
			Proxy: http.ProxyFromEnvironment,
		},
	}
}
