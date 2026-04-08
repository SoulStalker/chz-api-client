package handlers

import (
	"github.com/SoulStalker/chz-api-client/internal/signer"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type CertsHandler struct {
	signer *signer.Client
}

func NewCertsHandler(s *signer.Client) *CertsHandler {
	return &CertsHandler{signer: s}
}

func (h *CertsHandler) Handle(c *fiber.Ctx) error {
	certs, err := h.signer.ListCertificates(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Ошибка получения сертификатов: " + err.Error())
	}
	c.Type("html")
	return views.Certs(certs).Render(c.Context(), c.Response().BodyWriter())
}
