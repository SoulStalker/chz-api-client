package handlers

import (
	"errors"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/session"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type PacksHandler struct {
	crpt     *crpt.Client
	sessions *session.Store
}

func NewPacksHandler(c *crpt.Client, sessions *session.Store) *PacksHandler {
	return &PacksHandler{crpt: c, sessions: sessions}
}

func (h *PacksHandler) token(c *fiber.Ctx) (string, bool) {
	sessionID := c.Cookies("crpt_session")
	if sessionID == "" {
		return "", false
	}
	return h.sessions.Get(sessionID)
}

func (h *PacksHandler) ShowPack(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")
	packCode := c.Params("code")
	invoiceNumber := c.Query("invoice_number", docID)

	cisInfos, err := h.crpt.GetCisInfo(c.Context(), token, []string{packCode})
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.PackDetail(docID, packCode, invoiceNumber, nil, err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}
	if len(cisInfos) == 0 {
		c.Type("html")
		return views.PackDetail(docID, packCode, invoiceNumber, nil, "КИ не найдено").Render(c.Context(), c.Response().BodyWriter())
	}

	c.Type("html")
	return views.PackDetail(docID, packCode, invoiceNumber, &cisInfos[0], "").Render(c.Context(), c.Response().BodyWriter())
}

func (h *PacksHandler) ShowGroup(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")
	packCode := c.Params("code")
	groupCode := c.Params("childCode")
	invoiceNumber := c.Query("invoice_number", docID)

	cisInfos, err := h.crpt.GetCisInfo(c.Context(), token, []string{groupCode})
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.GroupDetail(docID, packCode, groupCode, invoiceNumber, nil, err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}
	if len(cisInfos) == 0 {
		c.Type("html")
		return views.GroupDetail(docID, packCode, groupCode, invoiceNumber, nil, "КИ не найдено").Render(c.Context(), c.Response().BodyWriter())
	}

	c.Type("html")
	return views.GroupDetail(docID, packCode, groupCode, invoiceNumber, &cisInfos[0], "").Render(c.Context(), c.Response().BodyWriter())
}
