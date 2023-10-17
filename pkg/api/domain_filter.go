package api

import "github.com/gofiber/fiber/v2"

func (w webhook) GetDomainFilter(ctx *fiber.Ctx) error {
	return ctx.JSON(w.provider.GetDomainFilter())
}
