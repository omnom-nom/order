package apiserver

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ServerImpl - Simple implementation of ApiServer
type ServerImpl struct {
	sync.Mutex

	// indicator of server status
	serverStatus ServerStatus

	// server status listener
	statusListener ServerStatusListener

	// parameters for running an HTTP server
	server http.Server

	// server shutdown timeout
	shutdownTimeout time.Duration
}

// ServerOpt ...
type ServerOpt func(*ServerImpl) error

// constructor --------------------------------------------------------

// ServerAddress ...
func ServerAddress(address string) ServerOpt {
	return func(srv *ServerImpl) error {
		srv.server.Addr = address
		return nil
	}
}

// ServerIP ...
func ServerIP(ip net.IP) ServerOpt {
	return func(srv *ServerImpl) error {
		if ip.To4() == nil {
			return errors.New("invalid api server IP: " + ip.String())
		}
		srv.server.Addr = ip.String() + ":" + strings.Split(srv.server.Addr, ":")[1]
		return nil
	}
}

// ServerPort ...
func ServerPort(port int) ServerOpt {
	return func(srv *ServerImpl) error {
		srv.server.Addr = strings.Split(srv.server.Addr, ":")[0] + ":" + strconv.Itoa(port)
		return nil
	}
}

// ServerCertificateFile ...
func ServerCertificateFile(certFile, keyFile string) ServerOpt {
	return func(srv *ServerImpl) error {
		cer, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}

		srv.server.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS11,
			Certificates: []tls.Certificate{cer},
		}

		return nil
	}
}

// ServerLogger ..
func ServerLogger(logger *log.Logger) ServerOpt {
	return func(srv *ServerImpl) error {
		srv.server.ErrorLog = logger
		return nil
	}
}

// ServerListener ...
func ServerListener(listener ServerStatusListener) ServerOpt {
	return func(srv *ServerImpl) error {
		srv.statusListener = listener
		return nil
	}
}

// New ...
func New(handler http.Handler, options ...ServerOpt) (Server, error) {
	impl := &ServerImpl{
		serverStatus:    Stopped,
		shutdownTimeout: DefaultShutdownTimeout,
	}

	impl.server.Handler = handler

	for _, option := range options {
		if err := option(impl); err != nil {
			return nil, err
		}
	}

	return impl, nil
}

// implementation -----------------------------------------------------

// StatusListener - Set listener for server status changes
func (srv *ServerImpl) StatusListener(listener ServerStatusListener) {
	srv.Lock()
	srv.statusListener = listener
	srv.Unlock()
}

// Endpoint - Query the details of the service endpoint
func (srv *ServerImpl) Endpoint() string {
	return "http" + "://" + srv.server.Addr
}

// IsRunning - Test if server is ready to accept requests
func (srv *ServerImpl) IsRunning() bool {
	srv.Lock()
	defer srv.Unlock()

	return srv.status() == Running
}

// IsStopped - Test if server has ceased listening for requests
func (srv *ServerImpl) IsStopped() bool {
	srv.Lock()
	defer srv.Unlock()

	return srv.status() == Stopped
}

// Query server status, make sure to acquire the lock first
func (srv *ServerImpl) status() ServerStatus {
	return srv.serverStatus
}

// Set server status, make sure to acquire the lock first
func (srv *ServerImpl) setStatus(status ServerStatus) {
	oldStatus := srv.serverStatus
	srv.serverStatus = status

	if srv.statusListener != nil {
		srv.statusListener(oldStatus, srv.serverStatus)
	}
}

// StartHTTP - Begin to listen for requests with HTTP protocol
func (srv *ServerImpl) StartHTTP() error {
	srv.Lock()
	defer srv.Unlock()

	if srv.status() != Stopped {
		return errors.New("api server is already running (or starting) on: " + srv.Endpoint())
	}

	// Run the server in a goroutine so that it doesn't block
	go func() {

		// Let's hope the listener will be started quickly, or
		// the running status that was set above will be less
		// than accurate
		// A reliable method would involve starting a goroutine
		// that would probe the server and updating the status
		// only after the first successul response, but that
		// would seem over-engineered
		srv.Lock()
		srv.setStatus(Running)
		srv.Unlock()

		if err := srv.server.ListenAndServe(); err != nil {
			if srv.server.ErrorLog != nil {
				srv.server.ErrorLog.Println(err)
			} else {
				log.Println("error: ", err)
			}
		}

		srv.Lock()
		srv.setStatus(Stopped)
		srv.Unlock()
	}()

	// start was initiated, status will be updated in a moment
	srv.setStatus(Starting)
	return nil
}

// StartHTTPS - Begin to listen for requests with HTTPS protocol
func (srv *ServerImpl) StartHTTPS() error {
	srv.Lock()
	defer srv.Unlock()

	if srv.status() != Stopped {
		return errors.New("api server is already running on: " + srv.Endpoint())
	}

	if srv.server.TLSConfig == nil {
		return errors.New("https api server can not start without SSL certificate and private key")
	}

	// Run the server in a goroutine so that it doesn't block
	go func() {

		// Let's hope the listener will be started quickly, or
		// the running status that was set above will be less
		// than accurate
		// A reliable method would involve starting a goroutine
		// that would probe the server and updating the status
		// only after the first successul response, but that
		// would seem over-engineered
		srv.Lock()
		srv.setStatus(Running)
		srv.Unlock()

		// TLS config is already initialized and verified
		if err := srv.server.ListenAndServeTLS("", ""); err != nil {
			if srv.server.ErrorLog != nil {
				srv.server.ErrorLog.Println(err)
			} else {
				log.Println("error: ", err)
			}
		}

		srv.Lock()
		srv.setStatus(Stopped)
		srv.Unlock()
	}()

	// start was initiated, status will be updated in a moment
	srv.setStatus(Starting)
	return nil
}

// Stop - End the listening process for requests
func (srv *ServerImpl) Stop() error {
	srv.Lock()
	defer srv.Unlock()

	if srv.status() == Stopped {
		return errors.New("api server is already stopped and not listening on: " + srv.Endpoint())
	}

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), srv.shutdownTimeout)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.server.Shutdown(ctx)

	// DO NOT set status to Stopped, this method only requerts to stop.
	// The status change will be reflected by the gorouting that listens
	// for incoming requests, once it exits the listining loop
	return nil
}
