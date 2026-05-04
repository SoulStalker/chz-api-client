package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/SoulStalker/chz-api-client/config"
	"github.com/SoulStalker/chz-api-client/internal/crpt"
	internalsigner "github.com/SoulStalker/chz-api-client/internal/signer"
	"github.com/SoulStalker/chz-api-client/internal/web/handlers"
	"github.com/SoulStalker/chz-api-client/internal/web/middleware"
	"github.com/gofiber/fiber/v2"
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

	signerClient, err := internalsigner.New(cfg.Signer.Addr)
	if err != nil {
		logger.Error("signer connect", "err", err)
		os.Exit(1)
	}
	defer signerClient.Close()

	crptClient := crpt.New(cfg.CRPT.BaseURL, signerClient)

	certsH := handlers.NewCertsHandler(signerClient)
	authH := handlers.NewAuthHandler(crptClient, cfg.CRPT.Thumbprint, cfg.CRPT.INN)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logger.Error("fiber error", "path", c.Path(), "err", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		},
	})

	app.Use(middleware.Logger(logger))

	app.Get("/", func(c *fiber.Ctx) error { return c.Redirect("/certs") })
	app.Get("/certs", certsH.Handle)
	app.Get("/auth", authH.Handle)
	app.Post("/auth", authH.Handle)

	logger.Info("starting server", "addr", cfg.Server.Addr)
	if err := app.Listen(cfg.Server.Addr); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
}
