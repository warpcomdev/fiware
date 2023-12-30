package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"cuelang.org/go/pkg/encoding/json"
	"github.com/warpcomdev/fiware/internal/keystone"
)

type loggingClient struct {
	verbosity int
	client    *http.Client
}

func (lc loggingClient) Do(req *http.Request) (*http.Response, error) {

	if lc.verbosity > 0 {
		fmt.Fprintln(os.Stderr, "-- performing request -- ")
		command := make([]string, 0, 16)
		for k, lv := range req.Header {
			for _, v := range lv {
				command = append(command, fmt.Sprintf("-H '%s: %s'", k, v))
			}
		}
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

		fmt.Fprintf(os.Stderr, "curl -X %s %s\n", req.Method, strings.Join(command, " "))
	}
	resp, err := lc.client.Do(req)
	if err == nil {
		if lc.verbosity > 1 {
			fmt.Fprintln(os.Stderr, "\n-- response headers -- ")
			headers := make([]string, 0, len(resp.Header)+1)
			headers = append(headers, fmt.Sprint(resp.Proto, resp.Status))
			for header, vals := range resp.Header {
				for _, val := range vals {
					headers = append(headers, fmt.Sprintf("%s: %s", header, val))
				}
			}
			fmt.Fprintln(os.Stderr, strings.Join(headers, "\n"))
		}
		if lc.verbosity > 2 && resp.Body != nil {
			fmt.Fprintln(os.Stderr, "\n-- response body -- ")
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			compacted, err := json.Indent(body, "", "  ")
			if err == nil {
				fmt.Fprintln(os.Stderr, string(compacted))
			} else {
				fmt.Fprintln(os.Stderr, string(body))
			}
			resp.Body = io.NopCloser(bytes.NewReader(body))
		}
	}
	return resp, err
}

func httpClient(verbosity int, timeout time.Duration) keystone.HTTPClient {
	return loggingClient{
		verbosity: verbosity,
		client:    _httpClient(timeout),
	}
}
