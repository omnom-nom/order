package apiserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// primitive way to avoid clashes with "real" central controller
	// which may be running in this host
	// LocalControllerPort ...
	LocalControllerPort = 14002
)

var testdata string

type Postdata struct {
	Data string `json:"Data"`
}

type Response struct {
	Error string `json:"Error"`
}

// TestAPIServer ...
func TestAPIServer(t *testing.T) {

	// step 1: make (empty) factory
	factory, err := FactoryForGorillaMux()
	if err != nil {
		log.Fatalf("failed to create API factory: %s", err)
	}

	// register middleware objects with factory
	factory.Default(MiddlewareLogger, Logger())

	const TestAPIUrlPrefix = "test"
	var routes = map[string][]Route{
		TestAPIUrlPrefix: {
			{Name: "GetData", Method: http.MethodGet, Path: "getdata", Handler: GetDataHandler},
			{Name: "PostData", Method: http.MethodPost, Path: "postdata", Handler: PostDataHandler},
			{Name: "DeleteData", Method: http.MethodDelete, Path: "deletedata", Handler: DeleteDataHandler},
		},
	}

	// create insecure (for http service) server
	mux, err := factory.Make(routes)
	if err != nil {
		log.Fatalf("failed to create API mux: %s", err)
	}

	server, err := New(mux, ServerAddress(fmt.Sprintf("%s:%d", "0.0.0.0", LocalControllerPort)))
	if err != nil {
		t.Fatalf("failed to create API server: %s", err)
	}

	if err = server.StartHTTP(); err != nil {
		t.Fatalf("failed to start HTTP API server: %s", err)
	}

	apiserverrunning := false
	for i := 0; i < 10; i++ {

		if server.IsRunning() {
			apiserverrunning = true
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !apiserverrunning {
		t.Error("Api server not running")
	}

	if resp, err := PostData("somedata"); err == nil {
		if len(resp.Error) > 0 {
			t.Error("PostData test failed! - Received error")
		}
	} else {
		t.Errorf("PostData test failed: %s", err)
	}

	if data, err := GetData(); err == nil {
		if strings.Compare(data, "somedata") != 0 {
			t.Errorf("GetData test failed! - data did not match: %s", data)
		}
	} else {
		t.Errorf("GetData test failed: %s", err)
	}

	if resp, err := DeleteData("somedata"); err == nil {
		if len(resp.Error) > 0 {
			t.Error("DeleteData test failed!")
		}
	} else {
		t.Errorf("DeleteData test failed: %s", err)
	}

	if err = server.Stop(); err != nil {
		t.Fatalf("failed to stop HTTP API server: %s", err)
	}
}

// PostDataHandler ...
func PostDataHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{}
	var pd Postdata

	if err := json.NewDecoder(r.Body).Decode(&pd); err != nil {
		log.Error(fmt.Sprintf("/postdata Bad Request Error: %s\n", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	testdata = pd.Data

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		log.Error(fmt.Sprintf("/postdata Error: %s\n", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetDataHandler ...
func GetDataHandler(w http.ResponseWriter, r *http.Request) {

	pd := &Postdata{}

	pd.Data = testdata

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pd); err != nil {
		log.Printf("/GetDataHandler Internal Error: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteDataHandler ...
func DeleteDataHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{}
	var pd Postdata

	if err := json.NewDecoder(r.Body).Decode(&pd); err != nil {
		log.Error(fmt.Sprintf("/deletedata Bad Request Error: %s\n", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.Compare(pd.Data, testdata) != 0 {
		resp.Error = "Data does not match"
	} else {
		testdata = ""
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		log.Error(fmt.Sprintf("/deleteData Error: %s\n", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func getControllerHTTPClient() (*http.Client, *http.Transport) {

	ctimeout := time.Duration(10) * time.Second
	tr := &http.Transport{
		Dial: func(network, addr string) (conn net.Conn, err error) {
			return net.DialTimeout("tcp", fmt.Sprintf("%s:%d", "localhost", LocalControllerPort), ctimeout)
		},
		MaxIdleConnsPerHost: 5,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(10) * time.Second,
	}

	return client, tr
}

// PostData ...
func PostData(data string) (*Response, error) {

	client, tr := getControllerHTTPClient()

	defer tr.CloseIdleConnections()

	pd := &Postdata{}

	pd.Data = data

	var jsondata []byte
	var err error

	if jsondata, err = json.Marshal(pd); err != nil {
		return nil, err
	}

	httpreq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:%d/test/postdata", "localhost", LocalControllerPort), bytes.NewReader(jsondata))
	if err == nil {

		resp, err1 := client.Do(httpreq)
		if err1 == nil {

			defer resp.Body.Close()
			var contents []byte
			if contents, err = ioutil.ReadAll(resp.Body); err == nil {

				resp := &Response{}

				if err = json.Unmarshal(contents, resp); err == nil {
					if len(resp.Error) == 0 {
						return resp, nil
					}
					return nil, errors.New(resp.Error)
				}

			}
			return nil, errors.New("Postdata Failed to read contents")
		}
		log.Debug("Postdata Ip Error: ", err1)
		return nil, err1
	}

	return nil, err
}

// DeleteData ...
func DeleteData(data string) (*Response, error) {

	client, tr := getControllerHTTPClient()

	defer tr.CloseIdleConnections()

	pd := &Postdata{}

	pd.Data = data

	var jsondata []byte
	var err error

	if jsondata, err = json.Marshal(pd); err != nil {
		return nil, err
	}

	if httpreq, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://%s:%d/test/deletedata", "localhost", LocalControllerPort), bytes.NewReader(jsondata)); err == nil {

		if resp, err := client.Do(httpreq); err == nil {

			defer resp.Body.Close()
			var contents []byte
			if contents, err = ioutil.ReadAll(resp.Body); err == nil {

				resp := &Response{}

				if err = json.Unmarshal(contents, resp); err == nil {
					if len(resp.Error) == 0 {
						return resp, nil
					}
					return nil, errors.New(resp.Error)
				}

			} else {

				return nil, errors.New("Deletedata Failed to read contents")
			}

		} else {
			log.Debug("Deletedata Ip Error: ", err)
			return nil, err
		}
	} else {
		return nil, err
	}

	return nil, errors.New("Deletedata failed - unknown reason")
}

// GetData ...
func GetData() (string, error) {

	client, tr := getControllerHTTPClient()

	defer tr.CloseIdleConnections()

	pd := &Postdata{}

	pd.Data = testdata

	if httpreq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/test/getdata", "localhost", LocalControllerPort), nil); err == nil {

		if resp, err := client.Do(httpreq); err == nil {

			defer resp.Body.Close()
			var contents []byte
			if contents, err = ioutil.ReadAll(resp.Body); err == nil {

				if err = json.Unmarshal(contents, pd); err == nil {
					return pd.Data, nil
				}
				return "", err
			}

		} else {
			log.Debug("Getdata Ip Error: ", err)
			return "", err
		}
	} else {
		return "", err
	}

	return "", errors.New("GetData Internal Error")
}
