package handlers

import (
	"log"
	"pelagica-collector/db"
	"pelagica-collector/models"
	"regexp"

	"github.com/gofiber/fiber/v3"
)

var (
	validUUID    = regexp.MustCompile(`^[0-9a-f-]{36}$`)
	validVersion = regexp.MustCompile(`^[0-9a-zA-Z.\-+]{1,32}$`)
)

type Handler struct {
	DB *db.DB
}

func (h *Handler) Ping(c fiber.Ctx) error {
	var req models.PingRequest

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("bad request")
	}

	if !validUUID.MatchString(req.InstanceID) ||
		!validVersion.MatchString(req.Version) {
		return c.Status(fiber.StatusBadRequest).SendString("invalid payload")
	}

	if err := h.DB.RecordPing(c.Context(), req.InstanceID, req.Version); err != nil {
		log.Printf("RecordPing error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("internal error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Stats(c fiber.Ctx) error {
	total, active, err := h.DB.GetStats(c.Context())
	if err != nil {
		log.Printf("GetStats error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("internal error")
	}

	return c.JSON(models.StatsResponse{
		TotalInstalls:   total,
		ActiveInstances: active,
	})
}
