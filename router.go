package routes

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Key is a type for context keys.
type Key string

// Router is a group of resolvers.
// Router implements Resolver and http.Handler interface.
type Router struct {
	prefix       string
	namespace    string
	defaultRoute *Route
	resolvers    map[string]Resolver
}

// Name returns router name (namespace).
func (router *Router) Name() string {
	return router.namespace
}

// ServeHTTP impelements http.Handler.ServeHTTP.
func (router *Router) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	router.Handle(response, request)
}

// Add adds new resolver to router.
// It's may replace existing resolver with same name.
func (router *Router) Add(resolver Resolver) {
	router.resolvers[resolver.Name()] = resolver
}

// Reverse returns URL path from matched resolver.
func (router *Router) Reverse(name string, parameters map[string]string) (path string, found bool) {
	parts := strings.Split(name, ":")

	if resolver, exists := router.resolvers[parts[0]]; exists {
		path, _ := resolver.Reverse(strings.Join(parts[1:], ":"), parameters)
		path = fmt.Sprintf("%v/%v", router.prefix, strings.TrimPrefix(path, "/"))
		return strings.Replace(path, "//", "/", 1), true
	}

	return "", false
}

// Resolve looking route by path.
func (router *Router) Resolve(path string) (*Route, bool) {
	if !strings.HasPrefix(path, router.prefix) {
		return nil, false
	}

	path = fmt.Sprintf("/%v", strings.Trim(strings.TrimPrefix(path, router.prefix), "/"))

	for _, route := range router.resolvers {
		if route, matched := route.Resolve(path); matched {
			return route, matched
		}
	}

	if router.defaultRoute != nil {
		return router.defaultRoute, true
	}

	return nil, false
}

// Handle looking for route by path and delegates request to handler.
// If route not found, Handle will write header http.StatusNotFound.
func (router *Router) Handle(response http.ResponseWriter, request *http.Request) {
	if route, found := router.Resolve(request.URL.Path); found {
		ctx := request.Context()

		for key, value := range route.GetGroups(request.URL.Path) {
			ctx = context.WithValue(ctx, Key(key), value)
		}

		route.handler.ServeHTTP(response, request.WithContext(ctx))
	} else {
		response.WriteHeader(http.StatusNotFound)
	}
}

// NewRouter creates new Router instance.
func NewRouter(prefix string, namespace string, defaultRoute *Route, resolvers ...Resolver) *Router {
	router := &Router{}
	router.prefix = fmt.Sprintf("/%v", strings.Trim(prefix, "/"))
	router.namespace = namespace
	router.defaultRoute = defaultRoute
	router.resolvers = make(map[string]Resolver)

	for _, resolver := range resolvers {
		router.Add(resolver)
	}

	return router
}
