package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
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

	t.Run("Listen", func(t *testing.T) {
		t.Parallel()

		go func() {
			err := app.Listen("5000")
			assert.NoError(t, err)
		}()

		time.Sleep(time.Second) // Allow server to start

		// Simulate SIGTERM signal to trigger shutdown
		process, _ := os.FindProcess(os.Getpid())
		err := process.Signal(syscall.SIGTERM)
		assert.NoError(t, err)
	})
}
