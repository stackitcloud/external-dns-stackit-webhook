package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	metricscollector "github.com/stackitcloud/external-dns-stackit-webhook/pkg/metrics"
	mockmetricscollector "github.com/stackitcloud/external-dns-stackit-webhook/pkg/metrics/mock"
)

func TestMetricsMiddleware(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	metricsCollector := mockmetricscollector.NewMockHttpApiMetrics(mockCtrl)

	// Expectations
	method := http.MethodGet
	path := "/"
	statusCode := http.StatusOK
	contentLength := float64(4)

	metricsCollector.EXPECT().CollectRequest(method, path, statusCode).Times(1)
	metricsCollector.EXPECT().CollectTotalRequests().Times(1)
	metricsCollector.EXPECT().CollectRequestResponseSize(method, path, contentLength).Times(1)
	metricsCollector.EXPECT().CollectRequestDuration(method, path, gomock.Any()).Times(1)

	app := setupTestMetrics(metricsCollector)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func setupTestMetrics(collector metricscollector.HttpApiMetrics) *fiber.App {
	app := fiber.New()
	app.Use(api.NewMetricsMiddleware(collector))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("test")
	})

	return app
}

func getTestMockMetricsCollector(ctrl *gomock.Controller) metricscollector.HttpApiMetrics {
	metricsCollector := mockmetricscollector.NewMockHttpApiMetrics(ctrl)

	metricsCollector.EXPECT().CollectRequest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	metricsCollector.EXPECT().CollectTotalRequests().AnyTimes()
	metricsCollector.EXPECT().CollectRequestResponseSize(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	metricsCollector.EXPECT().CollectRequestDuration(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	metricsCollector.EXPECT().Collect400TotalRequests().AnyTimes()
	metricsCollector.EXPECT().Collect500TotalRequests().AnyTimes()

	return metricsCollector
}
