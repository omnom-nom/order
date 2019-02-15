package apiserver

import (
	"net/http"
	"time"
)

const (
	// MiddlewareLogger ...
	MiddlewareLogger = "middleware:logger"
	// MiddlewareLeader ...
	MiddlewareLeader = "middleware:on_leader"
	// MiddlewareMd5Signature ...
	MiddlewareMd5Signature = "middleware:md5signature"
	// MiddlewareHostNotRegisreted ...
	MiddlewareHostNotRegisreted = "middleware:hostIsRegistered"
	// MiddlewareAuthorization ...
	MiddlewareAuthorization = "middleware:requestIsAuthorized"
	// MiddlewareRedirect ...
	MiddlewareRedirect = "middleware:requestIsRedirected"
	// MiddlewareDbStatus ...
	MiddlewareDbStatus = "middleware:checkdbstatus"
)

// Route defines a REST API endpoint
type Route struct {

	// Descriptive name of the the route
	Name string

	// Accepted HTTP method for this route
	Method string

	// URL expression for this route
	Path string

	// invoke this handler after all middleware processing
	Handler http.HandlerFunc

	// Names of middleware objects to include when processing this route
	Include []string

	// Names of middleware objects to exclude when processing this route
	Exclude []string
}

// ServiceFactory produces handlers for API Servers
type ServiceFactory interface {

	// Register middleware for all requests without exceptions
	Always(name string, middleware Middleware)

	// Register middleware for default use when making service handler
	Default(name string, middleware Middleware)

	// Register middleware that can be used when making service handler
	Available(name string, middleware Middleware)

	// Main method to make a service handler
	Make(routes map[string][]Route) (http.Handler, error)
}

// Server is a component that exposes http or https service
type Server interface {

	// Query the details of the service endpoint
	Endpoint() string

	// Install a listener for server status changes
	StatusListener(ServerStatusListener)

	// Test if server is ready to accept requests
	IsRunning() bool

	// Test if server has ceased listening for requests
	IsStopped() bool

	// Begin to listen and process requests with HTTP protocol
	StartHTTP() error

	// Begin to listen and process requests with HTTPS protocol
	StartHTTPS() error

	// End the listening process for requests
	Stop() error
}

// ServerStatus is the status of the Server
type ServerStatus uint

const (
	// Stopped ...
	Stopped ServerStatus = iota
	// Starting ...
	Starting
	// Running ...
	Running
	//DefaultShutdownTimeout ...
	DefaultShutdownTimeout = time.Second * 10
)

// ServerStatusListener receives updates for server status changes
type ServerStatusListener func(statusOld, statusNew ServerStatus)

// Middleware - Standard middleware interface definition
//
// ServeHTTP should yield to the next middleware in the chain by invoking
// the next http.HandlerFunc passed in. If the Handler writes to the
// ResponseWriter, the next http.HandlerFunc should not be invoked.
type Middleware interface {
	ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

// MiddlewareFunc - Adapter to allow the use of ordinary functions as middleware
type MiddlewareFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

// ServeHTTP ...
func (m MiddlewareFunc) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	m(rw, r, next)
}
