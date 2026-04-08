package handlers

import (
	"github.com/SoulStalker/chz-api-client/internal/diadoc"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type OrgsHandler struct {
	diadoc *diadoc.Client
	store  *session.Store
}

func NewOrgsHandler(d *diadoc.Client, store *session.Store) *OrgsHandler {
	return &OrgsHandler{diadoc: d, store: store}
}

func (h *OrgsHandler) Handle(c *fiber.Ctx) error {
	orgs, err := h.diadoc.GetMyOrganizations()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Ошибка получения организаций: " + err.Error())
	}
	return views.Orgs(orgs).Render(c.Context(), c.Response().BodyWriter())
}

func (h *OrgsHandler) Select(c *fiber.Ctx) error {
	boxId := c.FormValue("box_id")
	if boxId == "" {
		return c.Status(fiber.StatusBadRequest).SendString("box_id required")
	}

	sess, err := h.store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("session error")
	}
	sess.Set("box_id", boxId)
	if err := sess.Save(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("session save error")
	}

	return c.Redirect("/docs")
}
