package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	api2 "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api"
	mock_provider "github.com/stackitcloud/external-dns-stackit-webhook/pkg/api/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func prepareTest(t *testing.T) (*gomock.Controller, *api2.PropertyValuesEqualsRequest, []byte) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	pve := &api2.PropertyValuesEqualsRequest{
		Name:     "test.name",
		Current:  "test.current",
		Previous: "test.previous",
	}

	body, err := json.Marshal(pve)
	assert.NoError(t, err)

	return ctrl, pve, body
}

func setupAppAndMocks(ctrl *gomock.Controller) (api2.Api, *mock_provider.MockProvider) {
	mockLogger := zap.NewNop()
	mockProvider := mock_provider.NewMockProvider(ctrl)
	mockMetricsCollector := getTestMockMetricsCollector(ctrl)

	app := api2.New(mockLogger, mockMetricsCollector, mockProvider)

	return app, mockProvider
}

func TestWebhook_PropertyValuesEquals(t *testing.T) {
	t.Parallel()
	ctrl, pve, body := prepareTest(t)

	t.Run("Provider returns true", func(t *testing.T) {
		t.Parallel()
		app, mockProvider := setupAppAndMocks(ctrl)
		mockProvider.EXPECT().PropertyValuesEqual(pve.Name, pve.Previous, pve.Current).Return(true).Times(1)

		req := httptest.NewRequest(http.MethodPost, "/propertyvaluesequals", bytes.NewReader(body))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody := &api2.PropertiesValuesEqualsResponse{}
		err = json.NewDecoder(resp.Body).Decode(respBody)
		assert.NoError(t, err)
		assert.Equal(t, true, respBody.Equals)
	})

	t.Run("Provider returns false", func(t *testing.T) {
		t.Parallel()
		app, mockProvider := setupAppAndMocks(ctrl)
		mockProvider.EXPECT().PropertyValuesEqual(pve.Name, pve.Previous, pve.Current).Return(false).Times(1)

		reqFalse := httptest.NewRequest(http.MethodPost, "/propertyvaluesequals", bytes.NewReader(body))
		reqFalse.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respFalse, err := app.Test(reqFalse)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, respFalse.StatusCode)

		respBodyFalse := &api2.PropertiesValuesEqualsResponse{}
		err = json.NewDecoder(respFalse.Body).Decode(respBodyFalse)
		assert.NoError(t, err)
		assert.Equal(t, false, respBodyFalse.Equals)
	})

	t.Run("Client send invalid data", func(t *testing.T) {
		t.Parallel()
		reqBad := httptest.NewRequest(http.MethodPost, "/propertyvaluesequals", bytes.NewReader([]byte(`{"bad":"request"}`)))
		reqBad.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		app, mockProvider := setupAppAndMocks(ctrl)
		mockProvider.EXPECT().PropertyValuesEqual("", "", "").Return(false).Times(1)

		respBad, err := app.Test(reqBad)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, respBad.StatusCode)
	})

	t.Run("Client send invalid JSON", func(t *testing.T) {
		t.Parallel()
		app, _ := setupAppAndMocks(ctrl)

		reqBad := httptest.NewRequest(http.MethodPost, "/propertyvaluesequals", bytes.NewReader([]byte(`{"wrong:"request"`)))
		reqBad.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		respBad, err := app.Test(reqBad)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, respBad.StatusCode)
	})
}
