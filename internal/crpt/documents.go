package crpt

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

type DocListParams struct {
	PG    string
	Input bool
	Limit int
}

func (c *Client) ListDocuments(ctx context.Context, token string, p DocListParams) (*model.DocListResponse, error) {
	if p.Limit == 0 {
		p.Limit = 50
	}

	var result model.DocListResponse
	start := time.Now()
	resp, err := c.http.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetQueryParam("pg", p.PG).
		SetQueryParam("input", fmt.Sprintf("%t", p.Input)).
		SetQueryParam("limit", fmt.Sprintf("%d", p.Limit)).
		SetResult(&result).
		Get("/api/v4/true-api/doc/list")
	if err != nil {
		return nil, fmt.Errorf("crpt doc/list: %w", err)
	}
	slog.Info("crpt request",
		"method", "GET",
		"url", "/api/v4/true-api/doc/list",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if resp.StatusCode() == 401 {
		return nil, ErrUnauthorized
	}
	if resp.IsError() {
		return nil, fmt.Errorf("crpt doc/list: status %d: %s", resp.StatusCode(), resp.Body())
	}
	return &result, nil
}

func (c *Client) GetDocumentInfo(ctx context.Context, token, docID, pg string) (*model.DocInfoResponse, error) {
	// API returns a JSON array; we take the first element.
	var results []model.DocInfoResponse
	start := time.Now()
	resp, err := c.http.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetQueryParam("pg", pg).
		SetQueryParam("body", "true").
		SetResult(&results).
		Get("/api/v4/true-api/doc/" + docID + "/info")
	if err != nil {
		return nil, fmt.Errorf("crpt doc/info: %w", err)
	}
	slog.Info("crpt request",
		"method", "GET",
		"url", "/api/v4/true-api/doc/"+docID+"/info",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if resp.StatusCode() == 401 {
		return nil, ErrUnauthorized
	}
	if resp.IsError() {
		return nil, fmt.Errorf("crpt doc/info: status %d: %s", resp.StatusCode(), resp.Body())
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("crpt doc/info: пустой ответ для документа %s", docID)
	}
	return &results[0], nil
}
