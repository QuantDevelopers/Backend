package metrics

import "net/http"

type MetricsClient interface {
	IncHTTPRequest(method, route, status string)
	ObserveHTTPRequestDuration(method, route string, duration float64)
	Handler() http.Handler
}
