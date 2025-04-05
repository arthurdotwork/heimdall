package heimdall

import "net/url"

type Route struct {
	OriginalPath   string
	Target         *url.URL
	Method         string
	Headers        map[string][]string
	AllowedHeaders []string
}

type Router struct {
	// Routes is a map that index the route by path and method.
	Routes map[string]map[string]*Route
}

func NewRouter(endpoints []EndpointConfig) (*Router, error) {
	routes := make(map[string]map[string]*Route)

	for _, endpoint := range endpoints {
		targetURL, err := url.Parse(endpoint.Target)
		if err != nil {
			return nil, err
		}

		if _, ok := routes[endpoint.Path]; !ok {
			routes[endpoint.Path] = make(map[string]*Route)
		}

		routes[endpoint.Path][endpoint.Method] = &Route{
			OriginalPath:   endpoint.Path,
			Target:         targetURL,
			Method:         endpoint.Method,
			Headers:        endpoint.Headers,
			AllowedHeaders: endpoint.AllowedHeaders,
		}
	}

	return &Router{
		Routes: routes,
	}, nil
}

func (r *Router) GetRoute(path string, method string) (*Route, bool) {
	methodRoutes, ok := r.Routes[path]
	if !ok {
		return nil, false
	}

	route, ok := methodRoutes[method]
	if !ok {
		return nil, false
	}

	return route, true
}
