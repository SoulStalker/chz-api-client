package handlers

import (
	"errors"
	"strconv"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type DocsHandler struct {
	crpt *crpt.Client
}

func NewDocsHandler(c *crpt.Client) *DocsHandler {
	return &DocsHandler{crpt: c}
}

func (h *DocsHandler) Handle(c *fiber.Ctx) error {
	token := c.Cookies("crpt_token")
	if token == "" {
		return c.Redirect("/auth")
	}

	pageNum := 0
	if s := c.Query("page_num"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			pageNum = n
		}
	}

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	status := c.Query("status")

	f := crpt.DocFilter{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Type:     "LP_ACCEPT_GOODS",
		Status:   status,
		PageNum:  pageNum,
		PageSize: 50,
	}

	docs, total, err := h.crpt.IncomingDocuments(c.Context(), token, f)
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			c.ClearCookie("crpt_token")
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.Documents(nil, 0, dateFrom, dateTo, status, err.Error()).Render(c.Context(), c.Response().BodyWriter())
	}

	c.Type("html")
	return views.Documents(docs, total, dateFrom, dateTo, status, "").Render(c.Context(), c.Response().BodyWriter())
}
