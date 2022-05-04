//go:build !windows

package main

import (
	"crypto/tls"
	"net/http"
	"time"
)

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			// TODO: Hacer esto configurable
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// Para entornos con VPNS restrictivas
			Proxy: http.ProxyFromEnvironment,
		},
	}
}
