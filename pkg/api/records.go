package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func (w webhook) Records(ctx *fiber.Ctx) error {
	records, err := w.provider.Records(ctx.UserContext())
	if err != nil {
		w.logger.Error("Error getting records", zap.String(logFieldError, err.Error()))

		return ctx.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	w.logger.Debug("returning records", zap.String("records", fmt.Sprintf("%v", records)))

	ctx.Response().Header.Set(varyHeader, contentTypeHeader)

	return ctx.JSON(records)
}
