package api

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

func (w webhook) AdjustEndpoints(ctx *fiber.Ctx) error {
	var pve []*endpoint.Endpoint
	err := ctx.BodyParser(&pve)
	if err != nil {
		w.logger.Error("Error parsing body", zap.String(logFieldError, err.Error()))
		ctx.Response().Header.Set(contentTypeHeader, contentTypePlaintext)

		return ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	w.logger.Debug("requesting adjust endpoints count", zap.Int("count", len(pve)))

	pve = w.provider.AdjustEndpoints(pve)

	ctx.Response().Header.Set(varyHeader, contentTypeHeader)
	ctx.Response().Header.Set(contentTypeHeader, string(mediaTypeVersion1))

	return ctx.JSON(pve)
}
