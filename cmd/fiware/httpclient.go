//go:build !windows

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/warpcomdev/fiware/internal/keystone"
)

type loggingClient struct {
	verbose bool
	client  *http.Client
}

func (lc loggingClient) Do(req *http.Request) (*http.Response, error) {

	if lc.verbose {
		log.Println(" -- performing request -- ")
		command := make([]string, 0, 16)
		for k, lv := range req.Header {
			for _, v := range lv {
				command = append(command, fmt.Sprintf("-H '%s: %s'", k, v))
			}
		}
		command = append(command)
		command = append(command, fmt.Sprintf("'%s'", req.URL))

		var (
			body []byte
			err  error
		)
		if req.Body != nil {
			if body, err = ioutil.ReadAll(req.Body); err != nil {
				return nil, err
			}
		}
		if body != nil {
			command = append(command, fmt.Sprintf("-d '%s'", string(body)))
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		log.Printf("curl -X %s %s\n", req.Method, strings.Join(command, " "))
	}
	return lc.client.Do(req)
}

func httpClient(verbose bool) keystone.HTTPClient {
	return loggingClient{
		verbose: verbose,
		client:  _httpClient(),
	}
}
