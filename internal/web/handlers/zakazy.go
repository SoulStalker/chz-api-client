package handlers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/model"
	"github.com/SoulStalker/chz-api-client/internal/repository"
	"github.com/SoulStalker/chz-api-client/internal/session"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

const zakazPageSize = 50

type ZakazHandler struct {
	repo     *repository.ZakazRepo
	crpt     *crpt.Client
	sessions *session.Store
}

func NewZakazHandler(repo *repository.ZakazRepo, crptClient *crpt.Client, sessions *session.Store) *ZakazHandler {
	return &ZakazHandler{repo: repo, crpt: crptClient, sessions: sessions}
}

func (h *ZakazHandler) token(c *fiber.Ctx) (string, bool) {
	sessionID := c.Cookies("crpt_session")
	if sessionID == "" {
		return "", false
	}
	return h.sessions.Get(sessionID)
}

func (h *ZakazHandler) List(c *fiber.Ctx) error {
	now := time.Now()
	dateFrom := now.AddDate(0, 0, -30)
	dateTo := now

	displayFrom := dateFrom.Format("2006-01-02")
	displayTo := dateTo.Format("2006-01-02")

	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			dateFrom = t
			displayFrom = v
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			dateTo = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
			displayTo = v
		}
	}

	inn := c.Query("inn")

	page := 0
	if v := c.QueryInt("page", 0); v > 0 {
		page = v
	}
	offset := page * zakazPageSize

	zakazy, total, err := h.repo.List(c.Context(), dateFrom, dateTo, inn, offset, zakazPageSize)
	if err != nil {
		c.Type("html")
		return views.Zakazy(nil, 0, 0, 0, err.Error(), displayFrom, displayTo, inn).Render(c.Context(), c.Response().BodyWriter())
	}

	c.Type("html")
	return views.Zakazy(zakazy, total, page, zakazPageSize, "", displayFrom, displayTo, inn).Render(c.Context(), c.Response().BodyWriter())
}

func (h *ZakazHandler) Show(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	kod := c.Params("kod")
	zakaz, lines, err := h.repo.Get(c.Context(), kod)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Заказ не найден")
	}

	var candidates []model.Document
	dateFrom := zakaz.DataPostavki.AddDate(0, 0, -30).UTC().Format("2006-01-02T15:04:05.000Z")
	dateTo := zakaz.DataPostavki.AddDate(0, 0, 30).UTC().Format("2006-01-02T15:04:05.000Z")

	result, crptErr := h.crpt.ListDocuments(c.Context(), token, model.DocListParams{
		PG:       "water",
		Input:    true,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Limit:    50,
	})
	if crptErr == nil {
		for _, doc := range result.Results {
			if doc.SenderInn == zakaz.PostINN && doc.Type == "UNIVERSAL_TRANSFER_DOCUMENT" {
				candidates = append(candidates, doc)
			}
		}
	} else if errors.Is(crptErr, crpt.ErrUnauthorized) {
		return c.Redirect("/auth")
	}

	// Параллельно загружаем строки каждого кандидата для отображения деталей.
	candidateDocs := make([]model.CandidateDoc, len(candidates))
	if len(candidates) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i, doc := range candidates {
			wg.Add(1)
			go func(idx int, d model.Document) {
				defer wg.Done()
				cd := model.CandidateDoc{Document: d}
				if info, err := h.crpt.GetDocumentInfo(ctx, token, d.Number, "water"); err == nil {
					cd.Products = info.Body.Products
				}
				mu.Lock()
				candidateDocs[idx] = cd
				mu.Unlock()
			}(i, doc)
		}
		wg.Wait()
	}

	c.Type("html")
	return views.ZakazDetail(zakaz, lines, candidateDocs, "").Render(c.Context(), c.Response().BodyWriter())
}

func (h *ZakazHandler) Match(c *fiber.Ctx) error {
	kod := c.Params("kod")
	crptDocID := c.FormValue("crpt_doc_id")
	if crptDocID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("crpt_doc_id обязателен")
	}
	if err := h.repo.SetCRPTDocID(c.Context(), kod, crptDocID); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.Redirect("/zakazy/" + kod)
}

func (h *ZakazHandler) Unmatch(c *fiber.Ctx) error {
	kod := c.Params("kod")
	if err := h.repo.ClearCRPTDocID(c.Context(), kod); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.Redirect("/zakazy/" + kod)
}
