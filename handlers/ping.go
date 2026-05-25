package handlers

import (
	"log"
	"pelagica-collector/db"
	"pelagica-collector/models"
	"regexp"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/gofiber/fiber/v3"
)

var (
	validUUID    = regexp.MustCompile(`^[0-9a-f-]{36}$`)
	validVersion = regexp.MustCompile(`^[0-9a-zA-Z.\-+]{1,32}$`)
)

type Handler struct {
	DB      *db.DB
	limiter *ipLimiter
}

func NewHandler(db *db.DB) *Handler {
	return &Handler{
		DB:      db,
		limiter: newIPLimiter(),
	}
}

type ipLimiter struct {
	mu      sync.Mutex
	clients map[string]*rate.Limiter
}

func newIPLimiter() *ipLimiter {
	l := &ipLimiter{clients: make(map[string]*rate.Limiter)}
	// Clean up old entries every hour
	go func() {
		for range time.Tick(time.Hour) {
			l.mu.Lock()
			l.clients = make(map[string]*rate.Limiter)
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.clients[ip]; !ok {
		// 1 ping per 10 minutes per IP
		l.clients[ip] = rate.NewLimiter(rate.Every(10*time.Minute), 1)
	}
	return l.clients[ip].Allow()
}

func (h *Handler) Ping(c fiber.Ctx) error {
	if !h.limiter.allow(c.IP()) {
		return c.Status(fiber.StatusTooManyRequests).SendString("rate limit exceeded")
	}

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
