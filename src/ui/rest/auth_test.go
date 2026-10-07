package rest

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dresar/gowanew/config"
	"github.com/dresar/gowanew/pkg/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthEndpoints(t *testing.T) {
	config.AppPIN = "280219"

	app := fiber.New()
	InitRestAuth(app)

	bodyInvalid, _ := json.Marshal(LoginRequest{PIN: "999999"})
	reqInvalid := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(bodyInvalid))
	reqInvalid.Header.Set("Content-Type", "application/json")
	respInvalid, err := app.Test(reqInvalid)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, respInvalid.StatusCode)

	bodyValid, _ := json.Marshal(LoginRequest{PIN: "280219"})
	reqValid := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(bodyValid))
	reqValid.Header.Set("Content-Type", "application/json")
	respValid, err := app.Test(reqValid)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respValid.StatusCode)

	var loginResp utils.ResponseData
	err = json.NewDecoder(respValid.Body).Decode(&loginResp)
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", loginResp.Code)

	resMap, ok := loginResp.Results.(map[string]any)
	require.True(t, ok)
	token, ok := resMap["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	reqStatus := httptest.NewRequest("GET", "/auth/status", nil)
	reqStatus.Header.Set("Authorization", "Bearer "+token)
	respStatus, err := app.Test(reqStatus)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respStatus.StatusCode)

	var statusResp utils.ResponseData
	err = json.NewDecoder(respStatus.Body).Decode(&statusResp)
	require.NoError(t, err)
	statusMap, ok := statusResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, statusMap["authenticated"])

	reqLogout := httptest.NewRequest("POST", "/auth/logout", nil)
	respLogout, err := app.Test(reqLogout)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respLogout.StatusCode)
}
