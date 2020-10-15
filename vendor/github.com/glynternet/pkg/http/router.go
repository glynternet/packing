package http

import (
	"log"

	"github.com/gorilla/mux"
)

// NewRouter creates a new Router and initialises it will all of the global generateRoutes
func NewRouter(rs []Route, log *log.Logger) (*mux.Router, error) {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range rs {
		handler := LoggingHandler(log, route.AppHandler, route.Name)
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
		log.Printf("Route registered: %+v", route)
	}
	return router, nil
}
