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
	mchd       string
}

func NewAuthHandler(c *crpt.Client, thumbprint, inn, mchd string) *AuthHandler {
	return &AuthHandler{crpt: c, thumbprint: thumbprint, inn: inn, mchd: mchd}
}

func (h *AuthHandler) Handle(c *fiber.Ctx) error {
	c.Type("html")
	if c.Method() == fiber.MethodGet {
		return views.Auth(h.thumbprint, h.inn, h.mchd, "", "").Render(c.Context(), c.Response().BodyWriter())
	}
	thumbprint := c.FormValue("thumbprint")
	if thumbprint == "" {
		thumbprint = h.thumbprint
	}
	inn := c.FormValue("inn")
	if inn == "" {
		inn = h.inn
	}
	mchd := c.FormValue("mchd")
	if mchd == "" {
		mchd = h.mchd
	}
	token, err := h.crpt.Authenticate(c.Context(), thumbprint, inn, mchd)
	if err != nil {
		return views.Auth(thumbprint, inn, mchd, "", err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}
	c.Cookie(&fiber.Cookie{
		Name:     "crpt_token",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
	})
	return views.Auth(thumbprint, inn, mchd, token, "").Render(c.Context(), c.Response().BodyWriter())
}
