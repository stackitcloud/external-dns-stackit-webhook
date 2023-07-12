package api

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func (w webhook) PropertyValuesEquals(ctx *fiber.Ctx) error {
	pve := PropertyValuesEqualsRequest{}
	err := ctx.BodyParser(&pve)
	if err != nil {
		w.logger.Error("Error parsing body", zap.String(logFieldError, err.Error()))
		ctx.Response().Header.Set(contentTypeHeader, contentTypePlaintext)

		return ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	w.logger.Debug(
		"requesting property values",
		zap.String("name", pve.Name),
		zap.String("current", pve.Current),
		zap.String("previous", pve.Previous),
	)
	valuesEqual := w.provider.PropertyValuesEqual(pve.Name, pve.Previous, pve.Current)

	resp := PropertiesValuesEqualsResponse{
		Equals: valuesEqual,
	}
	ctx.Response().Header.Set(varyHeader, contentTypeHeader)
	ctx.Response().Header.Set(contentTypeHeader, string(mediaTypeVersion1))

	return ctx.JSON(resp)
}
