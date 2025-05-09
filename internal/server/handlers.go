package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"github.com/zachmann/tip/internal/config"
	"github.com/zachmann/tip/pkg"
)

var _tip *pkg.TIP

func tip() *pkg.TIP {
	if _tip == nil {
		_tip = pkg.NewTokenProxy(config.Get().TIP)
	}
	return _tip
}

func handleRemoteIntrospection(ctx *fiber.Ctx) error {
	var req pkg.TokenIntrospectionRequest
	if err := errors.WithStack(ctx.BodyParser(&req)); err != nil {
		errRes := pkg.TIPError{
			ErrorCode:        "invalid_request",
			ErrorDescription: fmt.Sprintf("cannot parse request body: %s", err.Error()),
			Status:           fiber.StatusBadRequest,
		}
		return ctx.Status(errRes.Status).JSON(errRes)
	}
	req.Authorization = ctx.Get("Authorization")
	res, err := tip().Introspect(req)
	if err != nil {
		var errRes pkg.TIPError
		if errors.As(err, &errRes) {
			return ctx.Status(errRes.Status).JSON(errRes)
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(
			pkg.TIPError{
				ErrorCode:        "internal_server_error",
				ErrorDescription: err.Error(),
			},
		)
	}
	return ctx.JSON(res)
}
