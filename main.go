package main

import (
	"context"
	_ "embed"
	"log"
	"os"
	"pelagica-collector/db"
	"pelagica-collector/handlers"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

//go:embed public/stats.html
var statsPage []byte

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4321"
	}
	return ":" + port
}

func getProxyHeader() string {
	proxyHeader := os.Getenv("PROXY_HEADER")
	if proxyHeader == "" {
		return fiber.HeaderXForwardedFor
	}
	return proxyHeader
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
		fiberCfg.ProxyHeader = getProxyHeader()
		fiberCfg.TrustProxy = true

		proxies := []string{"127.0.0.1", "::1"}

		if extra := os.Getenv("TRUSTED_PROXIES"); extra != "" {
			for _, p := range strings.Split(extra, ",") {
				proxies = append(proxies, strings.TrimSpace(p))
			}
		}

		fiberCfg.TrustProxyConfig = fiber.TrustProxyConfig{
			Proxies: proxies,
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

	app.Get("/", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.Send(statsPage)
	})
	app.Post("/ping", h.Ping)
	app.Get("/stats", h.Stats)

	log.Fatal(app.Listen(getPort()))
}
