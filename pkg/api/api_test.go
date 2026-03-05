package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
)

func TestApi(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	mockLogger := zap.NewNop()
	mockProvider := mock_provider.NewMockProvider(ctrl)
	mockMetricsCollector := getTestMockMetricsCollector(ctrl)

	app := api.New(mockLogger, mockMetricsCollector, mockProvider)

	t.Run("Test", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
