package apiserver

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	// ConnectTimeout ...
	ConnectTimeout = 1 * time.Second
	// RequestTimeout ...
	RequestTimeout = 5 * time.Second
)

// benchmark client funcs ---------------------------------------------

// BenchmarkApiServer_Version ...
func BenchmarkApiServer_Version(b *testing.B) {
	// client := makeHTTPClient(b)
	client, req := makeHTTPClient(b), makeHTTPGetRequest(b, "/v1/version")
	for i := 0; i < b.N; i++ {
		// callAPIServer(b, client, makeHTTPGetRequest(b, "/v1/version"))
		callAPIServer(b, client, req)
	}
}

// BenchmarkApiServer_Healthcheck ...
func BenchmarkApiServer_Healthcheck(b *testing.B) {
	// client := makeHTTPClient(b)
	client, req := makeHTTPClient(b), makeHTTPGetRequest(b, "/v1/healthcheck")
	for i := 0; i < b.N; i++ {
		// callAPIServer(b, client, makeHTTPGetRequest(b, "/v1/healthcheck"))
		callAPIServer(b, client, req)
	}
}

// BenchmarkApiServer_BindingsByNwtworkAndHost ...
func BenchmarkApiServer_BindingsByNwtworkAndHost(b *testing.B) {
	testNetwork, testInstance := "midtier", "sharad-instance-15.c.gpf-dev.internal"
	// client := makeHTTPClient(b)
	client, req := makeHTTPClient(b),
		makeHTTPGetRequest(b, fmt.Sprintf("/v1/bindingsbynetworkandhost/%s/%s",
			testNetwork, testInstance))
	for i := 0; i < b.N; i++ {
		// callAPIServer(b, client, makeHTTPGetRequest(b, fmt.Sprintf("/v1/bindingsbynetworkandhost/%s/%s",
		// 	                   			testNetwork, testInstance)))
		callAPIServer(b, client, req)
	}
}

// benchmark helper functions -----------------------------------------

func callAPIServer(b *testing.B, client *http.Client, req *http.Request) {
	b.Logf("callAPIServer: N=%d", b.N)
	for i := 0; i < b.N; i++ {
		resp, err := client.Do(req)
		if err != nil {
			b.Errorf("http request [method=%s, url=%s] failed: %s", req.Method, req.URL, err)
			continue
		}

		// var contents []byte
		defer resp.Body.Close()
		if _, err = ioutil.ReadAll(resp.Body); err != nil {
			b.Errorf("failed to read http response to [method=%s, url=%s]: %s", req.Method, req.URL, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			b.Errorf("http response to [method=%s, url=%s] is not success: code=%d, status=%s",
				req.Method, req.URL, resp.StatusCode, resp.Status)
			continue
		}
	}
}

func controllerAddress(b *testing.B) (addr string) {
	addr = os.Getenv("CC_ADDRESS")
	if addr == "" {
		b.Fatal("failed to create HTTP client, central controller address is not set in env")
	}
	return
}

func makeHTTPGetRequest(b *testing.B, url string) *http.Request {
	return makeHTTPRequest(b, "GET", url)
}

func makeHTTPRequest(b *testing.B, method, url string) *http.Request {
	addr := fmt.Sprintf("http://%s%s", controllerAddress(b), url)

	req, err := http.NewRequest(method, addr, nil)
	if err != nil {
		b.Fatalf("failed to create HTTP request to: %s, error = %s", addr, err)
	}

	return req
}

func makeHTTPClient(b *testing.B) *http.Client {
	return &http.Client{
		Timeout: RequestTimeout,
		Transport: &http.Transport{
			// DisableKeepAlives:	true,
			// MaxIdleConnsPerHost: 2,
			Dial: func(network, addr string) (conn net.Conn, err error) {
				// b.Logf("dialer: network=%s, addr=%s", network, addr)
				return net.DialTimeout("tcp", addr, ConnectTimeout)
			},
		},
	}
}
