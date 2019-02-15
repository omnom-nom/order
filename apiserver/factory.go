package apiserver

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

// --------------------------------------------------------------------

// API service factory to create HTTP Handler that uses GorillaMux
type gorillaMuxFactory struct {

	// middleware that is unconditionally and always added
	always map[string]Middleware

	// middleware that is added by default unless explicitly excluded
	defaults map[string]Middleware

	// middleware that can be added by explicit request in route definition
	available map[string]Middleware
}

// FactoryForGorillaMux ...
func FactoryForGorillaMux() (ServiceFactory, error) {
	return &gorillaMuxFactory{
		always:    make(map[string]Middleware),
		defaults:  make(map[string]Middleware),
		available: make(map[string]Middleware),
	}, nil
}

// implementation -----------------------------------------------------

// Always ...
func (f *gorillaMuxFactory) Always(name string, middleware Middleware) {
	f.always[name] = middleware
}

// Default ...
func (f *gorillaMuxFactory) Default(name string, middleware Middleware) {
	f.defaults[name] = middleware
}

// Available ...
func (f *gorillaMuxFactory) Available(name string, middleware Middleware) {
	f.available[name] = middleware
}

// Make ...
func (f *gorillaMuxFactory) Make(routeMap map[string][]Route) (http.Handler, error) {

	// 1. Create router
	router := mux.NewRouter()

	// 2. Prepare middleware objects that are always included
	alwaysHandlers := []negroni.Handler{}
	for _, middleware := range f.always {
		alwaysHandlers = append(alwaysHandlers, middleware)
	}

	always := negroni.New(alwaysHandlers...)

	// 3. Add URL prefix (if provided)
	routerWithPrefix := router
	for urlPrefix, routes := range routeMap {
		routerWithPrefix = router.PathPrefix("/" + urlPrefix).Subrouter().StrictSlash(true)
		routerWithPrefix.NotFoundHandler = always.With(negroni.Wrap(http.HandlerFunc(NotFoundHandler)))

		routerWithPrefix.HandleFunc("/", APIListingHandler)

		router.Path(urlPrefix).Handler(
			always.With(negroni.Wrap(routerWithPrefix)),
		)

		router.NotFoundHandler = always.With(negroni.Wrap(http.HandlerFunc(NotFoundHandler)))

		// to fill up default values, and to work on private copy of the routes
		updatedRoutes := updateRoutes(routes)

		// 4. Register the routes and their handlers
		for _, route := range updatedRoutes {

			subrouter := routerWithPrefix.Path("/").Subrouter().StrictSlash(true)
			if strings.Compare(route.Name, "Apis") == 0 {
				const dir= "/milkyway/swagger-ui/"
				router.PathPrefix("/api/").Handler(http.StripPrefix("/api/", http.FileServer(http.Dir(dir))))
			} else {
				subrouter.HandleFunc("/"+route.Path, route.Handler).Methods(route.Method).Name(route.Name)
			}

			excluded := make(map[string]interface{})
			for _, name := range route.Exclude {
				excluded[name] = struct{}{}
			}

			included := make(map[string]interface{})
			for _, name := range route.Include {
				if excluded[name] == nil {
					included[name] = struct{}{}
				}
			}

			middlewares := []negroni.Handler{}

			// add default middleware objects, minus the excluded ones
			for name, middleware := range f.defaults {
				if excluded[name] == nil {
					middlewares = append(middlewares, middleware)
				}
			}

			// add requested middleware objects, minus the default and excluded ones
			for _, name := range route.Include {
				if excluded[name] != nil {
					log.Errorf("middleware [%s] is present in <included> and <excluded> list, skipping", name)
					continue
				}

				if f.always[name] != nil {
					log.Warnf("middleware [%s] is unconditionally included, skipping", name)
					continue
				}

				if f.defaults[name] != nil {
					log.Warnf("middleware [%s] is included by default, skipping", name)
					continue
				}

				middleware, found := f.available[name]
				if !found {
					log.Errorf("middleware [%s] is not registered, skipping", name)
					continue

				}

				middlewares = append(middlewares, middleware)
			}

			middlewares = append(middlewares, negroni.Wrap(subrouter))

			routerWithPrefix.Path("/" + route.Path).Handler(always.With(middlewares...)).Methods(route.Method)
		}
	}

	// done
	return router, nil
}

// NotFoundHandler ...
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("rest api = %s %s, http=%d, proto=%s, sender=%s, agent=%s",
		r.Method, r.URL, http.StatusNotFound, r.Proto, r.RemoteAddr, r.UserAgent())

	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"Error": "API Not Supported"}`))
}

// APIListingHandler ...
func APIListingHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: walk over all registered handlers and produce the API listing
	w.WriteHeader(http.StatusNotFound)
}

// make a copy of the routes array to making changes to read-only data
func updateRoutes(routes []Route) []Route {

	routes2 := make([]Route, len(routes))
	copy(routes2, routes)

	for i := range routes {
		// set default method
		if routes[i].Method == "" {
			routes2[i].Method = http.MethodGet
		}

		// manually copy the arrays in lieu of deep copy
		routes2[i].Include = routes[i].Include[:]
		routes2[i].Exclude = routes[i].Exclude[:]
	}

	return routes2
}
