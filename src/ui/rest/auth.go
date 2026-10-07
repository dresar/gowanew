package rest

import (
	"strings"

	"github.com/dresar/gowanew/pkg/utils"
	"github.com/dresar/gowanew/ui/rest/middleware"
	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct{}

type LoginRequest struct {
	PIN string `json:"pin"`
}

func InitRestAuth(app fiber.Router) {
	handler := &AuthHandler{}
	app.Post("/auth/login", handler.Login)
	app.Get("/auth/status", handler.Status)
	app.Post("/auth/logout", handler.Logout)
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  fiber.StatusBadRequest,
			Code:    "INVALID_REQUEST",
			Message: "Format permintaan tidak valid",
		})
	}

	pin := strings.TrimSpace(req.PIN)
	if !middleware.ValidatePIN(pin) {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  fiber.StatusUnauthorized,
			Code:    "UNAUTHORIZED",
			Message: "PIN tidak valid",
		})
	}

	token := middleware.GenerateSessionToken()
	c.Cookie(&fiber.Cookie{
		Name:     "gowa_session",
		Value:    token,
		Path:     "/",
		SameSite: "Lax",
		MaxAge:   30 * 24 * 3600,
	})

	return c.JSON(utils.ResponseData{
		Status:  fiber.StatusOK,
		Code:    "SUCCESS",
		Message: "Login berhasil",
		Results: fiber.Map{
			"token": token,
		},
	})
}

func (h *AuthHandler) Status(c fiber.Ctx) error {
	authenticated := middleware.IsRequestAuthenticated(c)
	return c.JSON(utils.ResponseData{
		Status:  fiber.StatusOK,
		Code:    "SUCCESS",
		Results: fiber.Map{
			"authenticated": authenticated,
		},
	})
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "gowa_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: "Lax",
	})
	return c.JSON(utils.ResponseData{
		Status:  fiber.StatusOK,
		Code:    "SUCCESS",
		Message: "Logout berhasil",
	})
}
