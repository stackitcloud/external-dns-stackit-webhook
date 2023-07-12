package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/plan"
)

func (w webhook) ApplyChanges(ctx *fiber.Ctx) error {
	var changes plan.Changes
	err := ctx.BodyParser(&changes)
	if err != nil {
		w.logger.Error("Error parsing body", zap.String(logFieldError, err.Error()))
		ctx.Response().Header.Set(contentTypeHeader, contentTypePlaintext)

		return ctx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	w.logger.Debug(
		"requesting apply changes",
		zap.String("create", fmt.Sprintf("%v", changes.Create)),
		zap.String("delete", fmt.Sprintf("%v", changes.Delete)),
		zap.String("updateNew", fmt.Sprintf("%v", changes.UpdateNew)),
		zap.String("updateOld", fmt.Sprintf("%v", changes.UpdateOld)),
		zap.String("updateNew", fmt.Sprintf("%v", changes.UpdateNew)),
	)

	err = w.provider.ApplyChanges(ctx.UserContext(), &changes)
	if err != nil {
		w.logger.Error("Error applying changes", zap.String(logFieldError, err.Error()))
		ctx.Response().Header.Set(contentTypeHeader, contentTypePlaintext)

		return ctx.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	ctx.Status(fiber.StatusNoContent)

	return nil
}
