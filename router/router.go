package router

import (
	"context"
	"net/http"
	"strings"
)

type Router struct {
	routes []route
}

type route struct {
	method   string
	pattern  string
	segments []string
	handler  http.HandlerFunc
}

type routeParamsContextKey struct{}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) HandleFunc(method string, pattern string, handler http.HandlerFunc) {
	r.routes = append(r.routes, route{
		method:   method,
		pattern:  pattern,
		segments: routeSegments(pattern),
		handler:  handler,
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	methodNotAllowed := false
	reqSegments := routeSegments(req.URL.Path)

	for _, route := range r.routes {
		params, ok := matchRoute(route.segments, reqSegments)
		if !ok {
			continue
		}
		if route.method != req.Method {
			methodNotAllowed = true
			continue
		}

		ctx := context.WithValue(req.Context(), routeParamsContextKey{}, params)
		route.handler(w, req.WithContext(ctx))
		return
	}

	if methodNotAllowed {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	http.NotFound(w, req)
}

func PathParam(req *http.Request, name string) string {
	params, ok := req.Context().Value(routeParamsContextKey{}).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}

func routeSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func matchRoute(pattern []string, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}

	params := map[string]string{}
	for i, segment := range pattern {
		if isRouteParam(segment) {
			params[strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")] = path[i]
			continue
		}
		if segment != path[i] {
			return nil, false
		}
	}

	return params, true
}

func isRouteParam(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") && len(segment) > 2
}
