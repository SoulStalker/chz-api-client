package crpt

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

type DocFilter struct {
	DateFrom string
	DateTo   string
	Type     string
	Status   string
	PageNum  int
	PageSize int
}

type docItem struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	SenderName       string `json:"senderName"`
	SenderINN        string `json:"senderInn"`
	ReceiverName     string `json:"receiverName"`
	ReceiverINN      string `json:"receiverInn"`
	DocDate          string `json:"docDate"`
	CreatedTimestamp int64  `json:"createdTimestamp"`
	TotalItems       int    `json:"totalItems"`
}

type docPageResp struct {
	Results []docItem `json:"results"`
	Total   int       `json:"total"`
}

// IncomingDocuments возвращает список документов по заданному фильтру.
// token — JWT, полученный через Authenticate.
func (c *Client) IncomingDocuments(ctx context.Context, token string, f DocFilter) ([]model.Document, int, error) {
	if f.PageSize == 0 {
		f.PageSize = 50
	}

	req := c.http.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetQueryParam("pageNum", strconv.Itoa(f.PageNum)).
		SetQueryParam("pageSize", strconv.Itoa(f.PageSize))

	if f.DateFrom != "" {
		req.SetQueryParam("dateFrom", f.DateFrom)
	}
	if f.DateTo != "" {
		req.SetQueryParam("dateTo", f.DateTo)
	}
	if f.Type != "" {
		req.SetQueryParam("type", f.Type)
	}
	if f.Status != "" {
		req.SetQueryParam("status", f.Status)
	}

	var result docPageResp
	req.SetResult(&result)

	start := time.Now()
	resp, err := req.Get("/api/v3/true-api/facade/doc")
	if err != nil {
		return nil, 0, fmt.Errorf("crpt facade/doc: %w", err)
	}
	slog.Info("crpt request",
		"method", "GET",
		"url", "/facade/doc",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
		"total", result.Total,
	)
	if resp.StatusCode() == 401 {
		return nil, 0, ErrUnauthorized
	}
	if resp.IsError() {
		return nil, 0, fmt.Errorf("crpt facade/doc: status %d: %s", resp.StatusCode(), resp.Body())
	}

	docs := make([]model.Document, len(result.Results))
	for i, d := range result.Results {
		docs[i] = model.Document{
			ID:               d.ID,
			Type:             d.Type,
			Status:           d.Status,
			SenderName:       d.SenderName,
			SenderINN:        d.SenderINN,
			ReceiverName:     d.ReceiverName,
			ReceiverINN:      d.ReceiverINN,
			DocDate:          d.DocDate,
			CreatedTimestamp: d.CreatedTimestamp,
			TotalItems:       d.TotalItems,
		}
	}
	return docs, result.Total, nil
}
