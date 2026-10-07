package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dresar/gowanew/config"
	"github.com/dresar/gowanew/pkg/utils"
	"github.com/gofiber/fiber/v3"
)

var (
	tokenSecretOnce sync.Once
	tokenSecret     []byte
)

func getSecretSalt() []byte {
	tokenSecretOnce.Do(func() {
		saltFile := "storages/session.key"
		if data, err := os.ReadFile(saltFile); err == nil && len(data) >= 16 {
			tokenSecret = data
			return
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err == nil {
			_ = os.MkdirAll("storages", 0755)
			_ = os.WriteFile(saltFile, b, 0600)
			tokenSecret = b
			return
		}
		tokenSecret = []byte("gowa-persistent-session-salt-v1")
	})
	return tokenSecret
}

func getEffectivePIN() string {
	expected := strings.TrimSpace(config.AppPIN)
	if data, err := os.ReadFile("storages/pin.txt"); err == nil {
		if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
			expected = trimmed
		}
	}
	if expected == "" {
		expected = "280219"
	}
	return expected
}

func getTokenSecret() []byte {
	salt := getSecretSalt()
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(getEffectivePIN()))
	return mac.Sum(nil)
}

func ValidatePIN(pin string) bool {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return false
	}
	expected := getEffectivePIN()
	return subtle.ConstantTimeCompare([]byte(pin), []byte(expected)) == 1
}

func GenerateSessionToken() string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, getTokenSecret())
	mac.Write([]byte(ts))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s.%s", ts, sig)
}

func VerifyToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	if ValidatePIN(token) {
		return true
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	issued := time.Unix(ts, 0)
	if time.Since(issued) > 30*24*time.Hour || time.Until(issued) > 5*time.Minute {
		return false
	}
	mac := hmac.New(sha256.New, getTokenSecret())
	mac.Write([]byte(parts[0]))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expectedSig)) == 1
}

func VerifyBasicAuth(authHeader string) bool {
	if !strings.HasPrefix(strings.ToLower(authHeader), "basic ") {
		return false
	}
	encoded := strings.TrimSpace(authHeader[6:])
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}
	username := parts[0]
	password := parts[1]
	if ValidatePIN(password) {
		return true
	}
	for _, cred := range config.AppBasicAuthCredential {
		credParts := strings.SplitN(cred, ":", 2)
		if len(credParts) == 2 {
			if subtle.ConstantTimeCompare([]byte(username), []byte(credParts[0])) == 1 &&
				subtle.ConstantTimeCompare([]byte(password), []byte(credParts[1])) == 1 {
				return true
			}
		}
	}
	return false
}

func IsRequestAuthenticated(c fiber.Ctx) bool {
	authHeader := c.Get(fiber.HeaderAuthorization)
	if authHeader != "" {
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token := strings.TrimSpace(authHeader[7:])
			if VerifyToken(token) {
				return true
			}
		}
		if VerifyBasicAuth(authHeader) {
			return true
		}
		if VerifyToken(authHeader) {
			return true
		}
	}
	if pinHeader := c.Get("X-PIN"); pinHeader != "" {
		if ValidatePIN(pinHeader) {
			return true
		}
	}
	if apiKeyHeader := c.Get("X-API-Key"); apiKeyHeader != "" {
		if VerifyToken(apiKeyHeader) {
			return true
		}
	}
	if cookie := c.Cookies("gowa_session"); cookie != "" {
		if VerifyToken(cookie) {
			return true
		}
	}
	if queryToken := c.Query("token"); queryToken != "" {
		if VerifyToken(queryToken) {
			return true
		}
	}
	if queryPin := c.Query("pin"); queryPin != "" {
		if ValidatePIN(queryPin) {
			return true
		}
	}
	if queryAuth := c.Query("authorization"); queryAuth != "" {
		clean := strings.ReplaceAll(queryAuth, " ", "+")
		if VerifyBasicAuth("Basic " + clean) {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	p := strings.TrimSpace(path)
	if config.AppBasePath != "" && strings.HasPrefix(p, config.AppBasePath) {
		p = strings.TrimPrefix(p, config.AppBasePath)
	}
	if p == "" {
		p = "/"
	}
	return p
}

func isPublicPath(c fiber.Ctx) bool {
	p := normalizePath(c.Path())
	isGetOrHead := c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead
	if strings.Contains(c.Get("Accept"), "text/html") && isGetOrHead {
		return true
	}
	if isGetOrHead && (p == "/" || p == "/connect" || p == "/login" || p == "/dashboard" || p == "/messaging" || p == "/scheduled" || p == "/settings" || p == "/misc" || p == "/account" || p == "/integrations" || p == "/webhook") {
		return true
	}
	if p == "/auth/login" || p == "/auth/status" || p == "/auth/logout" {
		return true
	}
	if strings.HasPrefix(p, "/chatwoot/webhook") {
		return true
	}
	if strings.HasPrefix(p, "/.well-known/") || strings.HasPrefix(p, "/oauth/") {
		return true
	}
	return false
}

func AuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		p := normalizePath(c.Path())
		if isPublicPath(c) {
			return c.Next()
		}

		if p == "/mcp" {
			if IsRequestAuthenticated(c) {
				return c.Next()
			}
			c.Set("WWW-Authenticate", `Bearer realm="GoWA MCP"`)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": fiber.Map{
					"code":    -32000,
					"message": "Unauthorized: GoWA MCP requires authentication via Bearer token, PIN, or Basic Auth",
				},
			})
		}

		if IsRequestAuthenticated(c) {
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  fiber.StatusUnauthorized,
			Code:    "UNAUTHORIZED",
			Message: "Autentikasi diperlukan. Silakan masukkan PIN.",
		})
	}
}
