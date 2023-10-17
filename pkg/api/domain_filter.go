package api

import (
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

func (w webhook) GetDomainFilter(ctx *fiber.Ctx) error {
	data, err := json.Marshal(w.provider.GetDomainFilter())
	if err != nil {
		return err
	}

	ctx.Set(varyHeader, contentTypeHeader)
	ctx.Set(contentTypeHeader, mediaTypeFormat)

	return ctx.Send(data)
}
