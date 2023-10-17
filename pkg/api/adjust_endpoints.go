package api

import (
	"fmt"

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

	pve = w.provider.AdjustEndpoints(pve)

	w.logger.Debug("adjusted endpoints", zap.String("endpoints", fmt.Sprintf("%v", pve)))

	ctx.Set(varyHeader, contentTypeHeader)

	return ctx.JSON(pve)
}
