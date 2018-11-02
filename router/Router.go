package router

import (
	"github.com/gorilla/mux"
	"net/http"
)

//initialises a new router and adds all declared routes
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true).UseEncodedPath()
	for _, route := range RoutesGroup.defaultRoutes {
		handler := route.HandlerFunc
		handler = LogTime(handler)
		handler = HandleError(handler)
		router.Methods(route.Method).Path(route.Pattern).Handler(handler)
	}
	for _, route := range RoutesGroup.authRoutes {
		handler := route.HandlerFunc
		handler = LogTime(handler)
		handler = HandleError(handler)
		handler = Validate(handler)
		router.Methods(route.Method).Path(route.Pattern).Handler(handler)
	}
	return router
}

type Route struct {
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}
type RouteGroup struct {
	defaultRoutes []Route
	authRoutes    []Route
}

var RoutesGroup = &RouteGroup{}

type Routes []Route

//adds a non-authenticated route to the router
func AddDef(route Route) {
	RoutesGroup.defaultRoutes = append(RoutesGroup.defaultRoutes, route)
}

//adds a route that requires authentication to the router
func AddAuth(route Route) {
	RoutesGroup.authRoutes = append(RoutesGroup.authRoutes, route)
}
