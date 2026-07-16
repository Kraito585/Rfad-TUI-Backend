package handler

import (
	"Rfad-TUI-Backend/core/pkg/response"
	"Rfad-TUI-Backend/internal/service"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel"
)

type DefaultHandler struct {
	srv     *service.DefaultService
	is_prod bool
}

func NewDefaultHandler(srv *service.DefaultService, is_prod bool) *DefaultHandler {
	return &DefaultHandler{srv: srv}
}

var handlerTracer = otel.Tracer("http-handler")

func (h *DefaultHandler) GetLatestUpdate(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.GetLatestUpdate")
	defer span.End()

	update, err := h.srv.GetLatestUpdate(ctx)
	if err != nil {
		// Если обновлений нет, это не ошибка сервера, отдаем 404
		if err.Error() == "обновления еще не были опубликованы" {
			return response.Error(c, fiber.StatusNotFound, "Обновления не найдены", nil)
		}

		return response.Error(c, fiber.StatusInternalServerError, "Ошибка получения обновления", err.Error())
	}

	// Отдаем JSON в стандартной обертке APIResponse
	return response.OK(c, update)
}
