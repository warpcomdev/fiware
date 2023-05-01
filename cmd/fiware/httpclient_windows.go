//go:build windows

package main

import (
	"crypto/tls"
	"net/http"
	"time"

	ieproxy "github.com/mattn/go-ieproxy"
)

func _httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// TODO: Hacer esto configurable
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// Para entornos con VPNS restrictivas
			Proxy: ieproxy.GetProxyFunc(),
		},
	}
}
