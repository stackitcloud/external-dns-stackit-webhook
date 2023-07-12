package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HttpApiMetrics is an interface that defines the methods that can be used to collect metrics.
//
//go:generate mockgen -destination=./mock/http_middleware.go github.com/stackitcloud/external-dns-stackit-webhook HttpApiMetrics
type HttpApiMetrics interface {
	// CollectTotalRequests increment the total requests for the api
	CollectTotalRequests()
	// Collect400TotalRequests increment the total requests for the api with 400 status code
	Collect400TotalRequests()
	// Collect500TotalRequests increment the total requests for the api with 500 status code
	Collect500TotalRequests()
	// CollectRequest increment the total requests for the api with the given method and path and status code
	CollectRequest(method, path string, statusCode int)
	// CollectRequestContentLength increment the total content length for the api with the given method and path
	CollectRequestContentLength(method, path string, contentLength float64)
	// CollectRequestResponseSize increment the total response size for the api with the given method and path
	CollectRequestResponseSize(method, path string, contentLength float64)
	// CollectRequestDuration observe the histogram of the duration of the requests for the api with the given method and path
	CollectRequestDuration(method, path string, duration float64)
}

// httpApiMetrics is a struct that implements the HttpApiMetrics interface.
type httpApiMetrics struct {
	httpRequestsTotal        prometheus.Counter
	http500Total             prometheus.Counter
	http400Total             prometheus.Counter
	httpRequest              *prometheus.CounterVec
	httpRequestContentLength *prometheus.CounterVec
	httpRequestResponseSize  *prometheus.CounterVec
	httpRequestDuration      *prometheus.HistogramVec
}

// CollectTotalRequests increment the total requests for the api.
func (h *httpApiMetrics) CollectTotalRequests() {
	h.httpRequestsTotal.Inc()
}

// Collect400TotalRequests increment the total requests for the api with 400 status code.
func (h *httpApiMetrics) Collect400TotalRequests() {
	h.http400Total.Inc()
}

// Collect500TotalRequests increment the total requests for the api with 500 status code.
func (h *httpApiMetrics) Collect500TotalRequests() {
	h.http500Total.Inc()
}

// CollectRequest increment the total requests for the api with the given method and path and status code.
func (h *httpApiMetrics) CollectRequest(method, path string, statusCode int) {
	status := strconv.Itoa(statusCode)
	h.httpRequest.WithLabelValues(method, path, status).Inc()
}

// CollectRequestContentLength increment the total content length for the api with the given method and path.
func (h *httpApiMetrics) CollectRequestContentLength(method, path string, contentLength float64) {
	h.httpRequestContentLength.WithLabelValues(method, path).Add(contentLength)
}

// CollectRequestResponseSize increment the total response size for the api with the given method and path.
func (h *httpApiMetrics) CollectRequestResponseSize(method, path string, contentLength float64) {
	h.httpRequestResponseSize.WithLabelValues(method, path).Add(contentLength)
}

// CollectRequestDuration observe the histogram of the duration of the requests for the api with the given method and path.
func (h *httpApiMetrics) CollectRequestDuration(method, path string, duration float64) {
	h.httpRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// NewHttpApiMetrics returns a new instance of httpApiMetrics.
func NewHttpApiMetrics() HttpApiMetrics {
	return &httpApiMetrics{
		httpRequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "http_total_requests_total",
			Help: "The total number of processed HTTP requests",
		}),
		http400Total: promauto.NewCounter(prometheus.CounterOpts{
			Name: "http_requests_4xx_total",
			Help: "The total number of processed HTTP requests with status 4xx",
		}),
		http500Total: promauto.NewCounter(prometheus.CounterOpts{
			Name: "http_requests_5xx_total",
			Help: "The total number of processed HTTP requests with status 5xx",
		}),
		httpRequest: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "The total number of processed http requests",
		}, []string{"method", "path", "status_code"}),
		httpRequestContentLength: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_content_length_total",
			Help: "The umber of bytes received in each http request",
		}, []string{"method", "path"}),
		httpRequestResponseSize: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_response_size_total",
			Help: "The number of bytes returned in each http request",
		}, []string{"method", "path"}),
		httpRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_requests_request_duration",
			Help:    "Percentiles of HTTP request latencies in seconds",
			Buckets: getBucketHttpMetrics(),
		}, []string{"method", "path"}),
	}
}

// getBucketHttpMetrics returns the buckets for the http metrics.
func getBucketHttpMetrics() []float64 {
	return []float64{
		0.005,
		0.01,
		0.015,
		0.02,
		0.025,
		0.03,
		0.05,
		0.07,
		0.09,
		0.1,
		0.2,
		0.3,
		0.4,
		0.5,
		0.7,
		0.9,
		1,
		1.5,
		2,
		3,
		5,
		7,
		10,
	}
}
