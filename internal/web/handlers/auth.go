package handlers

import (
	"bytes"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/session"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	crpt       *crpt.Client
	sessions   *session.Store
	thumbprint string
	inn        string
	mchd       string
}

func NewAuthHandler(c *crpt.Client, sessions *session.Store, thumbprint, inn, mchd string) *AuthHandler {
	return &AuthHandler{crpt: c, sessions: sessions, thumbprint: thumbprint, inn: inn, mchd: mchd}
}

func (h *AuthHandler) Handle(c *fiber.Ctx) error {
	if c.Method() == fiber.MethodGet {
		c.Type("html")
		return views.Auth(h.thumbprint, h.inn, h.mchd, "", "", false, "").Render(c.Context(), c.Response().BodyWriter())
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
	showCurl := c.FormValue("show_curl") == "on"

	token, curlCmd, err := h.crpt.Authenticate(c.Context(), thumbprint, inn, mchd)
	curl := ""
	if showCurl {
		curl = curlCmd
	}

	if err != nil {
		c.Type("html")
		return views.Auth(thumbprint, inn, mchd, "", err.Error(), showCurl, curl).Render(c.Context(), c.Response().BodyWriter())
	}

	// Храним JWT в памяти, в cookie кладём только короткий UUID сессии.
	// Это обходит ограничение браузера на размер cookie (4 KB).
	sessionID := h.sessions.Set(token)

	var buf bytes.Buffer
	if err := views.Auth(thumbprint, inn, mchd, token, "", showCurl, curl).Render(c.Context(), &buf); err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     "crpt_session",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   36000,
		HTTPOnly: true,
		SameSite: "Lax",
	})
	c.Type("html")
	return c.Send(buf.Bytes())
}
