package apiserver

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// LoggingResponseWriter - Proxy for http.ResponseWriter to capture and save the status code
//
// [Go] Capturing the HTTP status code from http.ResponseWriter
// ref: http://ndersson.me/post/capturing_status_code_in_net_http/
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewLoggingResponseWriter ...
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{w, http.StatusOK}
}

// WriteHeader - Save the status code and delegate the call
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Logging middleware to record the start and end of the request processing
type logger struct {
	tag string
}

// LoggerOpt ...
type LoggerOpt func(*logger)

// LoggerTag ...
func LoggerTag(tag string) LoggerOpt {
	return func(logger *logger) { logger.tag = tag }
}

// Logger - Make a simple logging middleware that does not support customization of logging format
func Logger(options ...LoggerOpt) Middleware {
	logger := &logger{}

	for _, opt := range options {
		opt(logger)
	}

	return logger
}

// ServeHTTP ...
func (l *logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	now := time.Now()
	lrw := NewLoggingResponseWriter(rw)

	// mark the start of request processing
	if l.tag != "" {
		log.Debugf("rest api > %s, method=%s, proto=%s, sender=%s, agent=%s",
			l.tag, r.Method, r.Proto, r.RemoteAddr, r.UserAgent())
	} else {
		log.Debugf("rest api > %s %s, proto=%s, sender=%s, agent=%s",
			r.Method, r.URL, r.Proto, r.RemoteAddr, r.UserAgent())
	}

	// delegate
	next(lrw, r)

	// mark the end of request processing
	elapsed := time.Now().Sub(now)

	if l.tag != "" {
		log.Debugf("rest api < %s, http=%d, method=%s, sender=%s",
			l.tag, lrw.statusCode, r.Method, r.RemoteAddr)
	} else {
		log.Debugf("rest api < %s %s, http=%d, sender=%s, elapsed=%s",
			r.Method, r.URL, lrw.statusCode, r.RemoteAddr, elapsed)
	}
}
