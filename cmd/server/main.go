package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/SoulStalker/chz-api-client/config"
	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/diadoc"
	internalsigner "github.com/SoulStalker/chz-api-client/internal/signer"
	"github.com/SoulStalker/chz-api-client/internal/web/handlers"
	"github.com/SoulStalker/chz-api-client/internal/web/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	fibermemory "github.com/gofiber/storage/memory/v2"
)

func main() {
	cfgPath := flag.String("config", "config/prod.yml", "path to config file")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	// Sign-service gRPC client
	signerClient, err := internalsigner.New(cfg.Signer.Addr)
	if err != nil {
		logger.Error("signer connect", "err", err)
		os.Exit(1)
	}
	defer signerClient.Close()

	// Diadoc HTTP client
	diadocClient := diadoc.New(cfg.Diadoc.BaseURL, cfg.Diadoc.ClientID)

	// CRPT client
	crptClient := crpt.New(cfg.CRPT.BaseURL, signerClient)
	_ = crptClient

	// Session store (in-memory)
	store := session.New(session.Config{
		Storage: fibermemory.New(),
	})

	// Handlers
	certsH := handlers.NewCertsHandler(signerClient)
	authH := handlers.NewAuthHandler(diadocClient, store)
	orgsH := handlers.NewOrgsHandler(diadocClient, store)
	docsH := handlers.NewDocumentsHandler(diadocClient, store)
	detailH := handlers.NewDetailHandler(diadocClient, store)

	// Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logger.Error("fiber error", "path", c.Path(), "err", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		},
	})

	app.Use(middleware.Logger(logger))

	app.Get("/", func(c *fiber.Ctx) error { return c.Redirect("/certs") })
	app.Get("/certs", certsH.Handle)
	app.Post("/auth", authH.Handle)
	app.Get("/orgs", orgsH.Handle)
	app.Post("/orgs/select", orgsH.Select)
	app.Get("/docs", docsH.Handle)
	app.Get("/docs/:messageId", detailH.Handle)

	logger.Info("starting server", "addr", cfg.Server.Addr)
	if err := app.Listen(cfg.Server.Addr); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
