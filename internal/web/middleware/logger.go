package middleware

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Logger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		traceID := c.Get("X-Request-ID")
		if traceID == "" {
			traceID = fmt.Sprintf("%x", rand.Int63())
		}
		c.Set("X-Request-ID", traceID)
		c.Locals("trace_id", traceID)

		err := c.Next()

		logger.Info("http request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
			"trace_id", traceID,
		)

		return err
	}
}
