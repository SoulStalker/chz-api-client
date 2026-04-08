package handlers

import (
	"github.com/SoulStalker/chz-api-client/internal/diadoc"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type AuthHandler struct {
	diadoc *diadoc.Client
	store  *session.Store
}

func NewAuthHandler(d *diadoc.Client, store *session.Store) *AuthHandler {
	return &AuthHandler{diadoc: d, store: store}
}

func (h *AuthHandler) Handle(c *fiber.Ctx) error {
	login := c.FormValue("login")
	password := c.FormValue("password")
	thumbprint := c.FormValue("thumbprint")

	c.Type("html")
	token, err := h.diadoc.Authenticate(login, password)
	if err != nil {
		return views.AuthError(err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}

	sess, err := h.store.Get(c)
	if err != nil {
		return views.AuthError("Ошибка сессии: " + err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}
	sess.Set("diadoc_token", token)
	sess.Set("thumbprint", thumbprint)
	if err := sess.Save(); err != nil {
		return views.AuthError("Ошибка сохранения сессии: " + err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}

	h.diadoc.SetToken(token)
	return c.Redirect("/orgs")
}
