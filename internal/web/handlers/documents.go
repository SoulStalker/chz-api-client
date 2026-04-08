package handlers

import (
	"github.com/SoulStalker/chz-api-client/internal/diadoc"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type DocumentsHandler struct {
	diadoc *diadoc.Client
	store  *session.Store
}

func NewDocumentsHandler(d *diadoc.Client, store *session.Store) *DocumentsHandler {
	return &DocumentsHandler{diadoc: d, store: store}
}

func (h *DocumentsHandler) Handle(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Redirect("/certs")
	}
	boxId, ok := sess.Get("box_id").(string)
	if !ok || boxId == "" {
		return c.Redirect("/orgs")
	}

	docs, err := h.diadoc.GetDocuments(boxId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Ошибка получения документов: " + err.Error())
	}
	c.Type("html")
	return views.Documents(docs, boxId).Render(c.Context(), c.Response().BodyWriter())
}
