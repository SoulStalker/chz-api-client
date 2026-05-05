package handlers

import (
	"errors"
	"strconv"
	"time"

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

func (h *DocsHandler) Handle(c *fiber.Ctx) error {
	sessionID := c.Cookies("crpt_session")
	if sessionID == "" {
		c.Type("html")
		return views.Documents(nil, 0, "", "", "", 0, 15,
			"Сессия не найдена — авторизуйтесь на странице /auth").Render(c.Context(), c.Response().BodyWriter())
	}

	token, ok := h.sessions.Get(sessionID)
	if !ok {
		c.ClearCookie("crpt_session")
		c.Type("html")
		return views.Documents(nil, 0, "", "", "", 0, 15,
			"Сессия истекла — авторизуйтесь повторно на странице /auth").Render(c.Context(), c.Response().BodyWriter())
	}

	now := time.Now()

	dateFrom := c.Query("date_from")
	if dateFrom == "" {
		dateFrom = now.AddDate(0, 0, -3).Format("2006-01-02")
	}
	dateTo := c.Query("date_to")
	if dateTo == "" {
		dateTo = now.Format("2006-01-02")
	}

	pageNum := 0
	if s := c.Query("page_num"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			pageNum = n
		}
	}

	pageSize := 15
	if s := c.Query("page_size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			switch n {
			case 15, 30, 50, 100:
				pageSize = n
			}
		}
	}

	status := c.Query("status")

	f := crpt.DocFilter{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Type:     "LP_ACCEPT_GOODS",
		Status:   status,
		PageNum:  pageNum,
		PageSize: pageSize,
	}

	docs, total, err := h.crpt.IncomingDocuments(c.Context(), token, f)
	if err != nil {
		errMsg := err.Error()
		if errors.Is(err, crpt.ErrUnauthorized) {
			h.sessions.Delete(sessionID)
			c.ClearCookie("crpt_session")
			errMsg = "Токен истёк или недействителен (401) — авторизуйтесь снова на /auth"
		}
		c.Type("html")
		return views.Documents(nil, 0, dateFrom, dateTo, status, pageNum, pageSize, errMsg).Render(c.Context(), c.Response().BodyWriter())
	}

	c.Type("html")
	return views.Documents(docs, total, dateFrom, dateTo, status, pageNum, pageSize, "").Render(c.Context(), c.Response().BodyWriter())
}
