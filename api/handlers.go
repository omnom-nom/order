package api

import (
	"fmt"
	"encoding/json"
	"net/http"
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
