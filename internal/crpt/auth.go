package crpt

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"
)

type authKeyResp struct {
	UUID string `json:"uuid"`
	Data string `json:"data"`
}

type signInResp struct {
	Token string `json:"token"`
}

// Authenticate выполняет двухшаговый OAuth через УКЭП (CAdES-BES attached).
// Шаг 1: GET /api/v3/auth/key → {uuid, data}
// Шаг 2: Sign(data) → base64 → POST /api/v3/auth/simpleSignIn → JWT
func (c *Client) Authenticate(ctx context.Context, thumbprint string) (string, error) {
	// Шаг 1 — получить случайную строку для подписи
	start := time.Now()
	var keyResp authKeyResp
	resp, err := c.http.R().
		SetContext(ctx).
		SetResult(&keyResp).
		Get("/api/v3/auth/key")
	if err != nil {
		return "", fmt.Errorf("crpt auth/key: %w", err)
	}
	slog.Info("crpt request",
		"method", "GET",
		"url", "/auth/key",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if resp.IsError() {
		return "", fmt.Errorf("crpt auth/key: status %d: %s", resp.StatusCode(), resp.Body())
	}

	// Шаг 2 — подписать data как есть (не декодировать из base64)
	signedBytes, err := c.signer.Sign(ctx, []byte(keyResp.Data), thumbprint)
	if err != nil {
		return "", fmt.Errorf("crpt sign: %w", err)
	}
	slog.Info("sign",
		"thumbprint", thumbprint,
		"payload_len", len(keyResp.Data),
		"signed_len", len(signedBytes),
	)

	signedB64 := base64.StdEncoding.EncodeToString(signedBytes)

	// Шаг 3 — отправить uuid + подписанные данные
	start = time.Now()
	var tokenResp signInResp
	resp, err = c.http.R().
		SetContext(ctx).
		SetBody(map[string]string{
			"uuid": keyResp.UUID,
			"data": signedB64,
		}).
		SetResult(&tokenResp).
		Post("/api/v3/auth/simpleSignIn")
	if err != nil {
		return "", fmt.Errorf("crpt simpleSignIn: %w", err)
	}
	slog.Info("crpt request",
		"method", "POST",
		"url", "/auth/simpleSignIn",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if resp.IsError() {
		return "", fmt.Errorf("crpt simpleSignIn: status %d: %s", resp.StatusCode(), resp.Body())
	}

	slog.Info("token received", "token_len", len(tokenResp.Token), "expires_in", "10h")
	return tokenResp.Token, nil
}
