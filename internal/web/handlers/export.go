package handlers

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/model"
	"github.com/gofiber/fiber/v2"
	pgx "github.com/jackc/pgx/v5"
)

func (h *DocsHandler) ExportDocumentXML(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")

	info, err := h.crpt.GetDocumentInfo(c.Context(), token, docID, h.currentPG(c))
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	var topCodes []string
	for _, p := range info.Body.Products {
		topCodes = append(topCodes, p.Code)
	}
	if len(topCodes) == 0 {
		topCodes = info.Body.CisesList
	}

	codes, err := h.collectAllCodes(c, token, topCodes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	var zakazKod string
	if h.zakazy != nil {
		if z, err := h.zakazy.FindByCRPTDoc(c.Context(), docID); err == nil {
			zakazKod = z.Kod
		} else if !errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("export: zakaz lookup failed", "err", err)
		}
	}

	export := model.ExportRoot{
		Version: "1.0",
		Document: model.ExportDocument{
			UPDNumber:    info.InvoiceNumber,
			UPDDate:      isoDateOnly(info.InvoiceDate),
			SenderINN:    info.SenderInn,
			SenderName:   info.SenderName,
			ReceiverINN:  info.ReceiverInn,
			ReceiverName: info.ReceiverName,
			Status:       info.Status,
			ZakazKod:     zakazKod,
			Codes:        codes,
		},
	}

	xmlBytes, err := xml.MarshalIndent(export, "", "  ")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	filename := fmt.Sprintf("upd-%s.xml", info.InvoiceNumber)
	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, err = c.Write(append([]byte(xml.Header), xmlBytes...))
	return err
}

const exportBatchSize = 100

// collectAllCodes traverses the packaging hierarchy via BFS and returns a flat
// list of all marking codes with parent references.
func (h *DocsHandler) collectAllCodes(c *fiber.Ctx, token string, topCodes []string) ([]model.ExportCode, error) {
	const maxDepth = 5

	var result []model.ExportCode

	// current maps code → parentCode (empty string for top-level)
	current := make(map[string]string, len(topCodes))
	for _, code := range topCodes {
		current[code] = ""
	}

	for depth := 0; len(current) > 0 && depth < maxDepth; depth++ {
		batch := make([]string, 0, len(current))
		for code := range current {
			batch = append(batch, code)
		}

		// Query in chunks to avoid hitting API body size limits.
		infoMap := make(map[string]model.CisInfo, len(batch))
		for i := 0; i < len(batch); i += exportBatchSize {
			end := i + exportBatchSize
			if end > len(batch) {
				end = len(batch)
			}
			chunk := batch[i:end]
			cisInfos, err := h.crpt.GetCisInfo(c.Context(), token, chunk)
			if err != nil {
				slog.Warn("export: cises/info chunk failed", "depth", depth, "offset", i, "err", err)
				continue
			}
			for _, ci := range cisInfos {
				infoMap[ci.CisKey] = ci
			}
		}

		next := make(map[string]string)
		for _, code := range batch {
			parent := current[code]
			ci := infoMap[code]
			result = append(result, model.ExportCode{
				Code:       code,
				Type:       ci.PackType,
				Parent:     parent,
				ChildCount: ci.ChildCount,
			})
			for _, childCode := range ci.ChildCodes {
				next[childCode] = code
			}
		}

		current = next
	}

	return result, nil
}

func isoDateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
