package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"

	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
)

func TestWebhook_Records(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	expectedRecords := []*endpoint.Endpoint{
		{
			DNSName:    "test.endpoint",
			RecordType: "A",
			Targets:    []string{"127.0.0.1"},
		},
	}

	t.Run("Provider returns records successfully", func(t *testing.T) {
		t.Parallel()

		mockLogger := zap.NewNop()
		mockProvider := mock_provider.NewMockProvider(ctrl)
		mockMetricsCollector := getTestMockMetricsCollector(ctrl)

		app := api.New(mockLogger, mockMetricsCollector, mockProvider)
		mockProvider.EXPECT().Records(gomock.Any()).Return(expectedRecords, nil).Times(1)

		req := httptest.NewRequest(http.MethodGet, "/records", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respRecords []*endpoint.Endpoint
		err = json.NewDecoder(resp.Body).Decode(&respRecords)
		assert.NoError(t, err)
		assert.Equal(t, expectedRecords, respRecords)
	})

	t.Run("Provider returns no records", func(t *testing.T) {
		t.Parallel()

		mockLogger := zap.NewNop()
		mockProvider := mock_provider.NewMockProvider(ctrl)
		mockMetricsCollector := getTestMockMetricsCollector(ctrl)

		app := api.New(mockLogger, mockMetricsCollector, mockProvider)
		mockProvider.EXPECT().Records(gomock.Any()).Return(nil, fmt.Errorf("error")).Times(1)

		reqErr := httptest.NewRequest(http.MethodGet, "/records", nil)

		respErr, err := app.Test(reqErr)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, respErr.StatusCode)
	})
}
