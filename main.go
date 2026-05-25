package main

import (
	"context"
	"log"
	"os"
	"pelagica-collector/db"
	"pelagica-collector/handlers"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4321"
	}
	return ":" + port
}

func main() {
	ctx := context.Background()

	database, err := db.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := database.Migrate(ctx); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	h := handlers.NewHandler(database)

	fiberCfg := fiber.Config{}
	if os.Getenv("BEHIND_PROXY") == "true" {
		fiberCfg.ProxyHeader = fiber.HeaderXForwardedFor
		fiberCfg.TrustProxy = true
		fiberCfg.TrustProxyConfig = fiber.TrustProxyConfig{
			Proxies: []string{"127.0.0.1", "::1"},
		}
	}

	app := fiber.New(fiberCfg)

	app.Use(func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		return c.Next()
	})
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} (${latency}) - ${ip}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	app.Post("/ping", h.Ping)
	app.Get("/stats", h.Stats)

	log.Fatal(app.Listen(getPort()))
}
