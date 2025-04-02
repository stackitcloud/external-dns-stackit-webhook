package api

import (
	"strings"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	metrics_collector "github.com/stackitcloud/external-dns-stackit-webhook/pkg/metrics"
)

// registerAt registers the metrics endpoint.
func registerAt(app *fiber.App, path string) {
	app.Get(path, adaptor.HTTPHandler(promhttp.Handler()))
}

func NewMetricsMiddleware(collector metrics_collector.HttpApiMetrics) fiber.Handler {
	return func(c *fiber.Ctx) error {
		started := time.Now()

		// Continue with the chain of middleware and handlers
		err := c.Next()
		if err != nil {
			return err
		}

		method := strings.Clone(c.Method())
		path := strings.Clone(c.Path())
		status := c.Response().StatusCode()

		collector.CollectRequest(method, path, status)
		collector.CollectTotalRequests()
		collectHttpStatusErrors(status, collector)
		collector.CollectRequestResponseSize(method, path, float64(len(c.Response().Body())))
		collector.CollectRequestDuration(method, path, time.Since(started).Seconds())

		return nil
	}
}

func collectHttpStatusErrors(status int, collector metrics_collector.HttpApiMetrics) {
	if status >= 400 && status < 500 {
		collector.Collect400TotalRequests()
	} else if status >= 500 && status < 600 {
		collector.Collect500TotalRequests()
	}
}
