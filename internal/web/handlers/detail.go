package handlers

import (
	"github.com/SoulStalker/chz-api-client/internal/diadoc"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type DetailHandler struct {
	diadoc *diadoc.Client
	store  *session.Store
}

func NewDetailHandler(d *diadoc.Client, store *session.Store) *DetailHandler {
	return &DetailHandler{diadoc: d, store: store}
}

func (h *DetailHandler) Handle(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err != nil {
		return c.Redirect("/certs")
	}
	boxId, ok := sess.Get("box_id").(string)
	if !ok || boxId == "" {
		return c.Redirect("/orgs")
	}

	messageId := c.Params("messageId")
	entityId := c.Query("entity_id")

	detail, err := h.diadoc.GetDocument(boxId, messageId, entityId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Ошибка получения документа: " + err.Error())
	}
	c.Type("html")
	return views.Detail(detail).Render(c.Context(), c.Response().BodyWriter())
}
