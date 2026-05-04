package handlers

import (
	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	crpt       *crpt.Client
	thumbprint string
	inn        string
}

func NewAuthHandler(c *crpt.Client, thumbprint, inn string) *AuthHandler {
	return &AuthHandler{crpt: c, thumbprint: thumbprint, inn: inn}
}

func (h *AuthHandler) Handle(c *fiber.Ctx) error {
	c.Type("html")
	if c.Method() == fiber.MethodGet {
		return views.Auth(h.thumbprint, h.inn, "", "").Render(c.Context(), c.Response().BodyWriter())
	}
	thumbprint := c.FormValue("thumbprint")
	if thumbprint == "" {
		thumbprint = h.thumbprint
	}
	inn := c.FormValue("inn")
	token, err := h.crpt.Authenticate(c.Context(), thumbprint, inn)
	if err != nil {
		return views.Auth(thumbprint, inn, "", err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}
	return views.Auth(thumbprint, inn, token, "").Render(c.Context(), c.Response().BodyWriter())
}
