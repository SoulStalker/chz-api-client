// crpt-explore: разведочный скрипт для получения JWT через УКЭП (CAdES-BES attached).
// Алгоритм: ListCerts → auth/key → Sign → simpleSignIn → проверочный запрос.
// Конфигурация берётся из файла config/prod.yml (флаг -config).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/SoulStalker/chz-api-client/config"
	"github.com/SoulStalker/chz-api-client/internal/crpt"
	internalsigner "github.com/SoulStalker/chz-api-client/internal/signer"
	"github.com/go-resty/resty/v2"
)

func main() {
	cfgPath := flag.String("config", "config/prod.yml", "путь к файлу конфигурации")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		slog.Error("загрузка конфига", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Подключиться к sign-service gRPC
	slog.Info("подключение к sign-service", "addr", cfg.Signer.Addr)
	signerClient, err := internalsigner.New(cfg.Signer.Addr)
	if err != nil {
		slog.Error("не удалось подключиться к sign-service", "err", err)
		os.Exit(1)
	}
	defer signerClient.Close()

	// 2. ListCertificates → показать список
	certs, err := signerClient.ListCertificates(ctx)
	if err != nil {
		slog.Error("ListCertificates", "err", err)
		os.Exit(1)
	}
	fmt.Println("\n=== Доступные сертификаты ===")
	for i, cert := range certs {
		fmt.Printf("[%d] thumbprint=%s subject=%q notAfter=%s\n", i+1, cert.Thumbprint, cert.Subject, cert.NotAfter)
	}
	fmt.Println()

	// Thumbprint из конфига; если не задан — берём первый из списка
	thumbprint := cfg.CRPT.Thumbprint
	if thumbprint == "" {
		if len(certs) == 0 {
			slog.Error("нет доступных сертификатов и thumbprint не задан в конфиге")
			os.Exit(1)
		}
		thumbprint = certs[0].Thumbprint
		slog.Info("thumbprint не задан в конфиге, используется первый сертификат", "thumbprint", thumbprint)
	}

	// 3–6. Authenticate: GET /auth/key → Sign → POST /auth/simpleSignIn → JWT
	crptClient := crpt.New(cfg.CRPT.BaseURL, signerClient)
	token, err := crptClient.Authenticate(ctx, thumbprint, cfg.CRPT.INN, cfg.CRPT.MCHD)
	if err != nil {
		slog.Error("аутентификация в ЧЗ", "err", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== JWT токен ===\nДлина: %d символов\nТокен: %s\n\n", len(token), token)

	// 7. Проверочный запрос с токеном
	verifyURL := cfg.CRPT.BaseURL + "/api/v3/true-api/facade/edo/documents"
	slog.Info("проверочный запрос", "url", verifyURL)

	start := time.Now()
	resp, err := resty.New().R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+token).
		SetHeader("Accept", "application/json").
		Get(verifyURL)
	if err != nil {
		slog.Error("проверочный запрос", "err", err)
		os.Exit(1)
	}
	slog.Info("crpt request",
		"method", "GET",
		"url", "/api/v3/true-api/facade/edo/documents",
		"status", resp.StatusCode(),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	fmt.Printf("=== Ответ проверочного запроса ===\nStatus: %d\n", resp.StatusCode())

	// Pretty-print JSON если получилось
	var pretty interface{}
	if json.Unmarshal(resp.Body(), &pretty) == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(string(resp.Body()))
	}
}
