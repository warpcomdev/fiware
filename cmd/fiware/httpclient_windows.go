//go:build windows

package main

import (
	"crypto/tls"
	"net/http"
	"time"

	ieproxy "github.com/mattn/go-ieproxy"
)

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			// TODO: Hacer esto configurable
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// Para entornos con VPNS restrictivas
			Proxy: ieproxy.GetProxyFunc(),
		},
	}
}
