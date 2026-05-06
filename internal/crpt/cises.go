package crpt

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

func (c *Client) GetCisInfo(ctx context.Context, token string, codes []string) ([]model.CisInfo, error) {
	var result []model.CisInfo
	start := time.Now()
	resp, err := c.http.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetBody(codes).
		SetResult(&result).
		Post("/api/v3/true-api/cises/info")
	if err != nil {
		return nil, fmt.Errorf("crpt cises/info: %w", err)
	}
	slog.Info("crpt request",
		"method", "POST",
		"url", "/api/v3/true-api/cises/info",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if resp.StatusCode() == 401 {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode() == 404 {
		slog.Warn("crpt cises/info 404", "codes", codes)
		return nil, fmt.Errorf("crpt cises/info: КИ не найдены")
	}
	if resp.IsError() {
		return nil, httpErr("POST", "/api/v3/true-api/cises/info", resp.StatusCode(), resp.Body())
	}
	return result, nil
}
