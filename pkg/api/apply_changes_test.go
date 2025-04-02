package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"

	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
)

func TestWebhook_ApplyChanges(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	changes := getValidPlanChanges()
	body, err := json.Marshal(changes)
	assert.NoError(t, err)

	t.Run("Provider returns records successfully", func(t *testing.T) {
		t.Parallel()

		mockLogger := zap.NewNop()
		mockProvider := mock_provider.NewMockProvider(ctrl)
		mockMetricsCollector := getTestMockMetricsCollector(ctrl)

		app := api.New(mockLogger, mockMetricsCollector, mockProvider)
		mockProvider.EXPECT().ApplyChanges(gomock.Any(), changes).Return(nil).Times(1)

		req := httptest.NewRequest(http.MethodPost, "/records", bytes.NewReader(body))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("Invalid data send by client", func(t *testing.T) {
		t.Parallel()

		mockLogger := zap.NewNop()
		mockProvider := mock_provider.NewMockProvider(ctrl)
		mockMetricsCollector := getTestMockMetricsCollector(ctrl)

		app := api.New(mockLogger, mockMetricsCollector, mockProvider)
		mockProvider.EXPECT().ApplyChanges(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		reqBad := httptest.NewRequest(http.MethodPost, "/records", bytes.NewReader([]byte(`{"bad":"request"}`)))
		reqBad.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respBad, err := app.Test(reqBad)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, respBad.StatusCode)
	})

	t.Run("Provider returns error", func(t *testing.T) {
		t.Parallel()

		mockLogger := zap.NewNop()
		mockProvider := mock_provider.NewMockProvider(ctrl)
		mockMetricsCollector := getTestMockMetricsCollector(ctrl)

		app := api.New(mockLogger, mockMetricsCollector, mockProvider)
		mockProvider.EXPECT().ApplyChanges(gomock.Any(), changes).Return(fmt.Errorf("test error")).Times(1)

		reqFail := httptest.NewRequest(http.MethodPost, "/records", bytes.NewReader(body))
		reqFail.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respFail, err := app.Test(reqFail)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, respFail.StatusCode)
	})

	t.Run("Client send invalid JSON", func(t *testing.T) {
		t.Parallel()

		mockLogger := zap.NewNop()
		mockProvider := mock_provider.NewMockProvider(ctrl)
		mockMetricsCollector := getTestMockMetricsCollector(ctrl)

		app := api.New(mockLogger, mockMetricsCollector, mockProvider)

		reqBad := httptest.NewRequest(http.MethodPost, "/records", bytes.NewReader([]byte(`{"wrong:"request"`)))
		reqBad.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respBad, err := app.Test(reqBad)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, respBad.StatusCode)
	})
}

func getValidPlanChanges() *plan.Changes {
	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    "test.create.com",
				RecordType: "A",
			},
		},
		Delete: []*endpoint.Endpoint{
			{
				DNSName:    "test.delete.com",
				RecordType: "A",
			},
		},
		UpdateOld: []*endpoint.Endpoint{
			{
				DNSName:    "test.updateold.com",
				RecordType: "A",
			},
		},
		UpdateNew: []*endpoint.Endpoint{
			{
				DNSName:    "test.updatenew.com",
				RecordType: "A",
			},
		},
	}

	return changes
}
