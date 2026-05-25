package main

import (
	"context"
	"log"
	"pelagica-collector/db"
	"pelagica-collector/handlers"

	"github.com/gofiber/fiber/v3"
)

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

	app := fiber.New()

	app.Post("/ping", h.Ping)
	app.Get("/stats", h.Stats)

	log.Fatal(app.Listen(":4000"))
}
