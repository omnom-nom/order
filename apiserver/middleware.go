package apiserver

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// --------------------------------------------------------------------
// 			Common middleware functions
// --------------------------------------------------------------------

// ServiceCallCounter keeps track of number and rate of service calls
type ServiceCallCounter func()

// ServiceGatekeeper checks if the request can be accepted for processing
type ServiceGatekeeper func(*http.Request) bool

// CrashHandler performs crash recovery
type CrashHandler func(http.ResponseWriter)

// ServiceRedirect redirect API call from follower CC to leader CC
type ServiceRedirect func(http.ResponseWriter, *http.Request)

// --------------------------------------------------------------------
// 		Middleware objects for the above interfaces
// --------------------------------------------------------------------

// Adaptor for service call monitors
type serviceCallMonitor struct {
	counter ServiceCallCounter
}

// NewServiceCallCounter ...
func NewServiceCallCounter(counter ServiceCallCounter) Middleware {
	return &serviceCallMonitor{counter}
}

// ServeHTTP ...
func (m *serviceCallMonitor) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	m.counter()
	next(rw, r)
}

// Adaptor for service status monitors
type serviceGatekeeper struct {
	gatekeeper ServiceGatekeeper

	responseCode    int
	contentType     string
	responseMessage []byte
	logMessage      string
}

// ServiceGatekeeperOpt defines functional options for ServiceGatekeeper
type ServiceGatekeeperOpt func(*serviceGatekeeper)

// ServiceGatekeeperResponse provides the response body to send when the request is not accepted
func ServiceGatekeeperResponse(response []byte) ServiceGatekeeperOpt {
	return func(m *serviceGatekeeper) { m.responseMessage = response }
}

// ServiceGatekeeperResponseCode provides HTTP response code to send when the request is not accepted
func ServiceGatekeeperResponseCode(responseCode int) ServiceGatekeeperOpt {
	return func(m *serviceGatekeeper) { m.responseCode = responseCode }
}

// ServiceGatekeeperContentType provides the content type of the response when the request is not accepted
func ServiceGatekeeperContentType(contentType string) ServiceGatekeeperOpt {
	return func(m *serviceGatekeeper) { m.contentType = contentType }
}

// ServiceGatekeeperLogMessage provides the message to log when the request is not accepted
func ServiceGatekeeperLogMessage(logMessage string) ServiceGatekeeperOpt {
	return func(m *serviceGatekeeper) { m.logMessage = logMessage }
}

// ServiceGatekeeperTextResponse provides the test response to send when the request is not accepted
func ServiceGatekeeperTextResponse(response string) ServiceGatekeeperOpt {
	return func(m *serviceGatekeeper) {
		m.contentType = "text/plain"
		m.responseMessage = []byte(response)
	}
}

// ServiceGatekeeperJSONResponse ...
func ServiceGatekeeperJSONResponse(response interface{}) ServiceGatekeeperOpt {
	return func(m *serviceGatekeeper) {
		respJSON, err := json.Marshal(response)
		if err != nil {
			log.Errorf("service gatekeeper failed to serialize JSON response: %s", err)

			ServiceGatekeeperTextResponse("<internal error: response is not JSON format>")(m)
			return
		}

		m.contentType = "application/json"
		m.responseMessage = respJSON
	}
}

// NewServiceGatekeeper ...
func NewServiceGatekeeper(gatekeeper ServiceGatekeeper, options ...ServiceGatekeeperOpt) Middleware {
	g := &serviceGatekeeper{
		gatekeeper:   gatekeeper,
		responseCode: http.StatusServiceUnavailable,
		logMessage:   "service is not available",
	}

	for _, opt := range options {
		opt(g)
	}

	return g
}

// ServeHTTP executes a check if the request can accepted for processing
func (g *serviceGatekeeper) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if !g.gatekeeper(r) {
		log.Warn("api request is not accepted: " + g.logMessage)

		rw.Header().Set("Content-Type", g.contentType)
		rw.WriteHeader(g.responseCode)
		if len(g.responseMessage) > 0 {
			rw.Write(g.responseMessage)
		}

		// request not allowed, no further processing
		return
	}

	// request is allowed, proceed
	next(rw, r)
}

// Adaptor for crash handlers
type crashHandler struct {
	handler CrashHandler
}

// NewCrashHandler creates CrashHandler middleware
func NewCrashHandler(handler CrashHandler) Middleware {
	return &crashHandler{handler}
}

// ServeHTTP ...
func (m *crashHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer m.handler(rw)
	next(rw, r)
}

type serviceRedirect struct {
	handler ServiceRedirect
}

// NewServiceRedirectHandler creates a redirect middleware
func NewServiceRedirectHandler(handler ServiceRedirect) Middleware {
	return &serviceRedirect{handler}
}

// ServeHTTP ...
func (m *serviceRedirect) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	m.handler(rw, r)
	next(rw, r)
}
