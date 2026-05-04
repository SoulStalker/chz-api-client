// crpt-explore: разведочный скрипт для получения JWT через УКЭП (CAdES-BES attached).
// Алгоритм: ListCerts → auth/key → Sign → simpleSignIn → проверочный запрос.
//
// Переменные окружения:
//
//	CRPT_BASE_URL   — базовый URL стенда (по умолчанию https://markirovka.crpt.ru)
//	CRPT_THUMBPRINT — SHA1 hex отпечаток сертификата
//	SIGNER_ADDR     — gRPC адрес sign-service (по умолчанию localhost:50051)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	internalsigner "github.com/SoulStalker/chz-api-client/internal/signer"
	"github.com/go-resty/resty/v2"
)

func main() {
	baseURL := flag.String("base-url", getenv("CRPT_BASE_URL", "https://markirovka.crpt.ru"), "базовый URL ЧЗ")
	thumbprint := flag.String("thumbprint", os.Getenv("CRPT_THUMBPRINT"), "SHA1 отпечаток сертификата")
	signerAddr := flag.String("signer-addr", getenv("SIGNER_ADDR", "localhost:50051"), "gRPC адрес sign-service")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Подключиться к sign-service gRPC
	slog.Info("подключение к sign-service", "addr", *signerAddr)
	signerClient, err := internalsigner.New(*signerAddr)
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

	if *thumbprint == "" {
		if len(certs) == 0 {
			slog.Error("нет доступных сертификатов")
			os.Exit(1)
		}
		*thumbprint = certs[0].Thumbprint
		slog.Info("CRPT_THUMBPRINT не задан, используется первый сертификат", "thumbprint", *thumbprint)
	}

	// 3–6. Authenticate: GET /auth/key → Sign → POST /auth/simpleSignIn → JWT
	crptClient := crpt.New(*baseURL, signerClient)
	token, err := crptClient.Authenticate(ctx, *thumbprint)
	if err != nil {
		slog.Error("аутентификация в ЧЗ", "err", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== JWT токен ===\nДлина: %d символов\nТокен: %s\n\n", len(token), token)

	// 7. Проверочный запрос с токеном
	verifyURL := *baseURL + "/api/v3/true-api/facade/edo/documents"
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

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
