package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestWebhook_AdjustEndpoints(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockLogger := zap.NewNop()
	mockProvider := mock_provider.NewMockProvider(ctrl)
	mockMetricsCollector := getTestMockMetricsCollector(ctrl)

	app := api.New(mockLogger, mockMetricsCollector, mockProvider)

	t.Run("Test provider returns records successfully", func(t *testing.T) {
		t.Parallel()

		endpoints := []*endpoint.Endpoint{
			{
				DNSName:    "test.com",
				RecordType: "A",
			},
		}
		mockProvider.EXPECT().AdjustEndpoints(endpoints).Return(endpoints).Times(1)

		body, err := json.Marshal(endpoints)
		assert.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/adjustendpoints", bytes.NewReader(body))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Test invalid data send by client", func(t *testing.T) {
		t.Parallel()

		reqBad := httptest.NewRequest(http.MethodPost, "/adjustendpoints", bytes.NewReader([]byte(`{"bad":"request"}`)))
		reqBad.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respBad, err := app.Test(reqBad)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, respBad.StatusCode)
	})

	t.Run("Client send invalid JSON", func(t *testing.T) {
		t.Parallel()

		reqBad := httptest.NewRequest(http.MethodPost, "/adjustendpoints", bytes.NewReader([]byte(`{"wrong:"request"`)))
		reqBad.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respBad, err := app.Test(reqBad)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, respBad.StatusCode)
	})
}
