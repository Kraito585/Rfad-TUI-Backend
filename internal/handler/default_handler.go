package handler

import (
	"Rfad-TUI-Backend/core/pkg/response"
	"Rfad-TUI-Backend/internal/model"
	"Rfad-TUI-Backend/internal/service"
	"encoding/json"

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

func (h *DefaultHandler) GetCommunityShaders(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.GetCommunityShaders")
	defer span.End()

	presets, err := h.srv.GetAllPresets(ctx)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Не удалось получить пресеты шейдеров", err.Error())
	}

	return response.OK(c, presets)
}

func (h *DefaultHandler) UploadCommunityShader(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.UploadCommunityShader")
	defer span.End()

	form, err := c.MultipartForm()
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Ошибка парсинга multipart формы", err.Error())
	}

	metaStrings := form.Value["metadata"]
	if len(metaStrings) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "Отсутствует поле metadata", "")
	}

	var meta model.UploadPresetMetadata
	if err := json.Unmarshal([]byte(metaStrings[0]), &meta); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Неверный формат JSON в metadata", err.Error())
	}

	mainFiles := form.File["file"]
	if len(mainFiles) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "Отсутствует основной архив (file)", "")
	}

	imageFiles := form.File["images"]
	modFiles := form.File["mod_files"]

	presetID, err := h.srv.UploadPreset(ctx, &meta, mainFiles[0], imageFiles, modFiles)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Ошибка сохранения пресета", err.Error())
	}

	return response.OK(c, fiber.Map{
		"success": true,
		"id":      presetID.String(),
	})
}

func (h *DefaultHandler) UploadConfig(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.UploadConfig")
	defer span.End()

	body := c.Body()

	err := h.srv.UploadConfig(ctx, body)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Ошибка сохранения конфига", err.Error())
	}

	return response.OK(c, fiber.Map{
		"success": true,
	})
}

func (h *DefaultHandler) DisableConfig(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.DisableConfig")
	defer span.End()

	var req model.SwitchConfig
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Неверный формат запроса", err.Error())
	}

	err := h.srv.DisableConfig(ctx, req.UUID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Ошибка отключения конфига", err.Error())
	}

	return response.OK(c, fiber.Map{
		"success": true,
	})
}

func (h *DefaultHandler) EnableConfig(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.EnableConfig")
	defer span.End()

	var req model.SwitchConfig
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Неверный формат запроса", err.Error())
	}

	err := h.srv.EnableConfig(ctx, req.UUID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Ошибка включение конфига", err.Error())
	}

	return response.OK(c, fiber.Map{
		"success": true,
	})
}

func (h *DefaultHandler) GetConfigs(c fiber.Ctx) error {
	ctx, span := handlerTracer.Start(c.Context(), "handler.GetConfigs")
	defer span.End()

	configs, err := h.srv.GetConfig(ctx)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Ошибка получения конфигов", err.Error())
	}
	return response.OK(c, fiber.Map{
		"success": true,
		"data":    configs,
	})
}
