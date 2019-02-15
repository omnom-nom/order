package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"time"

	"github.com/omnom-nom/apiserver"
)

const (
	// APIServerStartupTimeout ...
	APIServerStartupTimeout = 5 * time.Second
	// APIServerStartupWaitPause ...
	APIServerStartupWaitPause = 500 * time.Millisecond
)


type Product struct {
	Name  string  `json:"Name"`
}


func HealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Println("healthcheck api")

	prod := &Product{Name: "chirag"}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prod); err != nil {

		fmt.Printf("/HostRegistrationStatus Internal Error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleCrash(w http.ResponseWriter) {
	crash := recover()
	if crash == nil {
		return
	}
	fmt.Println(crash)
}

func main() {

	// step 1: make (empty) factory
	factory, err := apiserver.FactoryForGorillaMux()
	if err != nil {
		fmt.Println(err)
		return
	}

	// register middleware objects with factory
	factory.Default(apiserver.MiddlewareLogger, apiserver.Logger())
	//factory.Always("crash-handler", apiserver.NewCrashHandler(handleCrash))

	const v1 = "v1"
	secureRoutes := map[string][]apiserver.Route{
		v1: {
			{
				Name:    "HealthCheck",
				Method:  http.MethodGet,
				Path:    "healthcheck",
				Handler: HealthCheck,
			},
		},
	}

	fmt.Println(secureRoutes)

	// create secure (for https service) server
	secureMux, err := factory.Make(secureRoutes)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(secureMux)

	httpServer, err := apiserver.New(secureMux, apiserver.ServerAddress(fmt.Sprintf("%s:%d", "0.0.0.0", 8080)))
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = httpServer.StartHTTP(); err != nil {
		fmt.Printf("failed to start HTTPS API server: %s", err)
		return
	}
	defer func() {
		if !httpServer.IsStopped() {
			if err := httpServer.Stop(); err != nil {
				errMsg := fmt.Sprintf("failed to stop HTTP server: %s", err)
				fmt.Println(errMsg)
			}
		}
	}()

	waitUntil := time.Now().Add(APIServerStartupTimeout)
	for waitUntil.After(time.Now()) {
		if httpServer.IsRunning() {
			break
		}
		if httpServer.IsStopped() {
			fmt.Printf("http server has stopped, can not continue")
			return
		}

		fmt.Println("waiting for api servers to start...")
		time.Sleep(APIServerStartupWaitPause)
	}

	if !httpServer.IsRunning() {
		fmt.Println("http server is not running")
		return
	}

	fmt.Println("http server is running", httpServer.IsRunning(), httpServer.Endpoint())

	for {
		time.Sleep(120 * time.Second)
		continue
	}
}
