package handlers

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/model"
	"github.com/SoulStalker/chz-api-client/internal/session"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

func errMsg(err error) string {
	switch {
	case errors.Is(err, crpt.ErrForbidden):
		return "Нет доступа — проверьте права сертификата"
	case errors.Is(err, crpt.ErrServiceUnavailable):
		return "Сервис временно недоступен, попробуйте позже"
	default:
		return err.Error()
	}
}

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

func (h *DocsHandler) currentPG(c *fiber.Ctx) string {
	if pg := c.Query("pg"); pg != "" && model.IsValidPG(pg) {
		c.Cookie(&fiber.Cookie{Name: "crpt_pg", Value: pg, Path: "/"})
		return pg
	}
	if pg := c.Cookies("crpt_pg"); model.IsValidPG(pg) {
		return pg
	}
	return "water"
}

func (h *DocsHandler) ListDocuments(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	pg := h.currentPG(c)
	isIncoming := strings.Contains(c.Path(), "incoming")

	now := time.Now().UTC()
	displayDateFrom := now.Add(-72 * time.Hour).Format("2006-01-02")
	displayDateTo := now.Format("2006-01-02")
	apiDateFrom := now.Add(-72 * time.Hour).Format("2006-01-02T15:04:05.000Z")
	apiDateTo := now.Format("2006-01-02T15:04:05.000Z")

	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			apiDateFrom = start.Format("2006-01-02T15:04:05.000Z")
			displayDateFrom = v
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999000000, time.UTC)
			apiDateTo = end.Format("2006-01-02T15:04:05.000Z")
			displayDateTo = v
		}
	}

	did := c.Query("did")
	ocv := c.Query("ordered_column_value")

	params := model.DocListParams{
		PG:                 pg,
		Input:              isIncoming,
		DateFrom:           apiDateFrom,
		DateTo:             apiDateTo,
		Limit:              50,
		DID:                did,
		OrderedColumnValue: ocv,
	}

	result, err := h.crpt.ListDocuments(c.Context(), token, params)
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		msg := errMsg(err)
		c.Type("html")
		return views.Documents(nil, false, msg, isIncoming, displayDateFrom, displayDateTo, "", "", pg, model.ProductGroups).Render(c.Context(), c.Response().BodyWriter())
	}

	var nextDID, nextOCV string
	if result.NextPage && len(result.Results) > 0 {
		last := result.Results[len(result.Results)-1]
		nextDID = last.Number
		nextOCV = last.DocDate
	}

	c.Type("html")
	return views.Documents(result.Results, result.NextPage, "", isIncoming, displayDateFrom, displayDateTo, nextDID, nextOCV, pg, model.ProductGroups).Render(c.Context(), c.Response().BodyWriter())
}

func (h *DocsHandler) ShowDocument(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")
	from := c.Query("from", "incoming")

	info, err := h.crpt.GetDocumentInfo(c.Context(), token, docID, h.currentPG(c))
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.DocumentDetail(docID, nil, nil, errMsg(err), from).Render(c.Context(), c.Response().BodyWriter())
	}

	cisInfoMap := make(map[string]model.CisInfo)
	var codes []string
	for _, p := range info.Body.Products {
		codes = append(codes, p.Code)
	}
	if len(codes) == 0 {
		codes = info.Body.CisesList
	}
	if len(codes) > 0 {
		cisInfos, ciErr := h.crpt.GetCisInfo(c.Context(), token, codes)
		if ciErr != nil {
			slog.Warn("cises/info failed, continuing without enrichment", "err", ciErr)
		} else {
			for _, ci := range cisInfos {
				cisInfoMap[ci.CisKey] = ci
			}
		}
	}

	c.Type("html")
	return views.DocumentDetail(docID, info, cisInfoMap, "", from).Render(c.Context(), c.Response().BodyWriter())
}
