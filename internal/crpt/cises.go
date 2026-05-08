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
		PackageType        string   `json:"packageType"`        // LEVEL1 | LEVEL2 | UNIT (fallback)
		GeneralPackageType string   `json:"generalPackageType"` // BOX | GROUP | UNIT (preferred)
		Child              []string `json:"child"`
	} `json:"cisInfo"`
}

// resolvePackType returns the canonical pack type (BOX/GROUP/UNIT).
// CRPT fills generalPackageType for containers (BOX/GROUP) but often leaves it
// empty for individual consumer units. packageType provides a fallback.
func resolvePackType(general, pkg string, hasChildren bool) string {
	if general != "" {
		return general
	}
	switch pkg {
	case "LEVEL1":
		return "BOX"
	case "LEVEL2":
		return "GROUP"
	}
	if !hasChildren {
		return "UNIT"
	}
	return pkg
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
		hasChildren := len(item.CisInfo.Child) > 0
		result = append(result, model.CisInfo{
			CisKey:     key,
			GTIN:       item.CisInfo.GTIN,
			PackType:   resolvePackType(item.CisInfo.GeneralPackageType, item.CisInfo.PackageType, hasChildren),
			ChildCount: len(item.CisInfo.Child),
			ChildCodes: item.CisInfo.Child,
		})
	}
	return result, nil
}
