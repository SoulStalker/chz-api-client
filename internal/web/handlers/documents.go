package handlers

import (
	"errors"
	"sort"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/session"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type DocsHandler struct {
	crpt     *crpt.Client
	sessions *session.Store
}

func NewDocsHandler(c *crpt.Client, sessions *session.Store) *DocsHandler {
	return &DocsHandler{crpt: c, sessions: sessions}
}

func (h *DocsHandler) token(c *fiber.Ctx) (string, bool) {
	sessionID := c.Cookies("crpt_session")
	if sessionID == "" {
		return "", false
	}
	return h.sessions.Get(sessionID)
}

func (h *DocsHandler) ListDocuments(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	result, err := h.crpt.ListDocuments(c.Context(), token, crpt.DocListParams{
		PG:    "water",
		Input: true,
		Limit: 50,
	})
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.Documents(nil, false, err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}

	docs := result.Results
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].DocDate > docs[j].DocDate
	})

	c.Type("html")
	return views.Documents(docs, result.NextPage, "").Render(c.Context(), c.Response().BodyWriter())
}

func (h *DocsHandler) ShowDocument(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")
	info, err := h.crpt.GetDocumentInfo(c.Context(), token, docID, "water")
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.DocumentDetail(nil, err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}

	c.Type("html")
	return views.DocumentDetail(info, "").Render(c.Context(), c.Response().BodyWriter())
}
