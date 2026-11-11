package metrics

import (
	"net/http"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusMetrics struct {
	metrics             *bootstrap.Metrics
	registry            *prometheus.Registry
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
}

func NewPrometheusMetrics(metrics *bootstrap.Metrics) *PrometheusMetrics {
	reg := prometheus.NewRegistry()

	reqTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.HTTPRequestsTotal.Name,
			Help: metrics.HTTPRequestsTotal.Help,
		},
		[]string{"method", "route", "status"},
	)

	reqDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metrics.HTTPRequestDuration.Name,
			Help:    metrics.HTTPRequestDuration.Help,
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	reg.MustRegister(reqTotal, reqDuration)

	return &PrometheusMetrics{
		metrics:             metrics,
		registry:            reg,
		httpRequestsTotal:   reqTotal,
		httpRequestDuration: reqDuration,
	}
}

func (pm *PrometheusMetrics) IncHTTPRequest(method, route, status string) {
	pm.httpRequestsTotal.WithLabelValues(method, route, status).Inc()
}

func (pm *PrometheusMetrics) ObserveHTTPRequestDuration(method, route string, duration float64) {
	pm.httpRequestDuration.WithLabelValues(method, route).Observe(duration)
}

func (pm *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(pm.registry, promhttp.HandlerOpts{})
}
