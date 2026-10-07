package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dresar/gowanew/config"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePIN(t *testing.T) {
	config.AppPIN = "280219"
	assert.True(t, ValidatePIN("280219"))
	assert.True(t, ValidatePIN(" 280219 "))
	assert.False(t, ValidatePIN("123456"))
	assert.False(t, ValidatePIN(""))
}

func TestGenerateAndVerifyToken(t *testing.T) {
	config.AppPIN = "280219"
	token := GenerateSessionToken()
	require.NotEmpty(t, token)
	assert.True(t, VerifyToken(token))
	assert.True(t, VerifyToken("280219"))
	assert.False(t, VerifyToken("invalid.token"))
	assert.False(t, VerifyToken(""))
}

func TestAuthMiddleware_ProtectedRoutes(t *testing.T) {
	config.AppPIN = "280219"

	app := fiber.New()
	app.Use(WebsocketQueryAuth())
	app.Use(AuthMiddleware())

	app.Get("/devices", func(c fiber.Ctx) error {
		return c.SendString("devices list")
	})
	app.Post("/mcp", func(c fiber.Ctx) error {
		return c.SendString(`{"jsonrpc":"2.0","result":"ok"}`)
	})

	reqNoAuth := httptest.NewRequest("GET", "/devices", nil)
	respNoAuth, err := app.Test(reqNoAuth)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, respNoAuth.StatusCode)

	reqBearerPin := httptest.NewRequest("GET", "/devices", nil)
	reqBearerPin.Header.Set("Authorization", "Bearer 280219")
	respBearerPin, err := app.Test(reqBearerPin)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respBearerPin.StatusCode)

	token := GenerateSessionToken()
	reqBearerToken := httptest.NewRequest("GET", "/devices", nil)
	reqBearerToken.Header.Set("Authorization", "Bearer "+token)
	respBearerToken, err := app.Test(reqBearerToken)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respBearerToken.StatusCode)

	reqXPin := httptest.NewRequest("GET", "/devices", nil)
	reqXPin.Header.Set("X-PIN", "280219")
	respXPin, err := app.Test(reqXPin)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respXPin.StatusCode)

	reqBasic := httptest.NewRequest("GET", "/devices", nil)
	reqBasic.SetBasicAuth("admin", "280219")
	respBasic, err := app.Test(reqBasic)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respBasic.StatusCode)

	reqCookie := httptest.NewRequest("GET", "/devices", nil)
	reqCookie.Header.Set("Cookie", "gowa_session="+token)
	respCookie, err := app.Test(reqCookie)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respCookie.StatusCode)

	reqQuery := httptest.NewRequest("GET", "/devices?token="+token, nil)
	respQuery, err := app.Test(reqQuery)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respQuery.StatusCode)

	reqMcpNoAuth := httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"jsonrpc":"2.0"}`))
	reqMcpNoAuth.Header.Set("Content-Type", "application/json")
	respMcpNoAuth, err := app.Test(reqMcpNoAuth)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, respMcpNoAuth.StatusCode)
	var mcpErr map[string]any
	err = json.NewDecoder(respMcpNoAuth.Body).Decode(&mcpErr)
	require.NoError(t, err)
	assert.Contains(t, mcpErr, "error")

	reqMcpAuth := httptest.NewRequest("POST", "/mcp", strings.NewReader(`{"jsonrpc":"2.0"}`))
	reqMcpAuth.Header.Set("Content-Type", "application/json")
	reqMcpAuth.Header.Set("Authorization", "Bearer 280219")
	respMcpAuth, err := app.Test(reqMcpAuth)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respMcpAuth.StatusCode)
}

func TestAuthMiddleware_PublicRoutes(t *testing.T) {
	config.AppPIN = "280219"

	app := fiber.New()
	app.Use(AuthMiddleware())

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("dashboard html")
	})
	app.Get("/login", func(c fiber.Ctx) error {
		return c.SendString("login html")
	})
	app.Get("/chats", func(c fiber.Ctx) error {
		return c.SendString("chats html")
	})

	reqRoot := httptest.NewRequest("GET", "/", nil)
	respRoot, err := app.Test(reqRoot)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respRoot.StatusCode)

	reqLogin := httptest.NewRequest("GET", "/login", nil)
	respLogin, err := app.Test(reqLogin)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respLogin.StatusCode)

	reqChatsHtml := httptest.NewRequest("GET", "/chats", nil)
	reqChatsHtml.Header.Set("Accept", "text/html,application/xhtml+xml")
	respChatsHtml, err := app.Test(reqChatsHtml)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respChatsHtml.StatusCode)
}
