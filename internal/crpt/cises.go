package crpt

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

// cisInfoAPIItem matches the actual CRPT v3 cises/info response structure.
type cisInfoAPIItem struct {
	CisInfo struct {
		RequestedCis       string   `json:"requestedCis"`
		Cis                string   `json:"cis"`
		GTIN               string   `json:"gtin"`
		GeneralPackageType string   `json:"generalPackageType"` // BOX | UNIT | GROUP
		Child              []string `json:"child"`              // child cis codes
	} `json:"cisInfo"`
}

func (c *Client) GetCisInfo(ctx context.Context, token string, codes []string) ([]model.CisInfo, error) {
	var raw []cisInfoAPIItem
	start := time.Now()
	resp, err := c.http.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetBody(codes).
		SetResult(&raw).
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

	result := make([]model.CisInfo, 0, len(raw))
	for _, item := range raw {
		key := item.CisInfo.Cis
		if key == "" {
			key = item.CisInfo.RequestedCis
		}
		result = append(result, model.CisInfo{
			CisKey:     key,
			GTIN:       item.CisInfo.GTIN,
			PackType:   item.CisInfo.GeneralPackageType,
			ChildCount: len(item.CisInfo.Child),
			ChildCodes: item.CisInfo.Child,
		})
	}
	return result, nil
}
