package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockLogger := zap.NewNop()
	mockProvider := mock_provider.NewMockProvider(ctrl)
	mockMetricsCollector := getTestMockMetricsCollector(ctrl)

	app := api.New(mockLogger, mockMetricsCollector, mockProvider)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
