package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	logger2 "github.com/zachmann/tip/internal/logger"
)

func addMiddlewares(s fiber.Router) {
	addRecoverMiddleware(s)
	addRequestIDMiddleware(s)
	addLoggerMiddleware(s)
	addHelmetMiddleware(s)
	addCompressMiddleware(s)
}

func addLoggerMiddleware(s fiber.Router) {
	s.Use(
		logger.New(
			logger.Config{
				Format:     "${time} ${ip} ${ua} ${latency} - ${status} ${method} ${path} ${locals:requestid}\n",
				TimeFormat: "2006-01-02 15:04:05",
				Output:     logger2.MustGetAccessLogger(),
			},
		),
	)
}

func addCompressMiddleware(s fiber.Router) {
	s.Use(compress.New())
}

func addRecoverMiddleware(s fiber.Router) {
	s.Use(recover.New())
}

func addHelmetMiddleware(s fiber.Router) {
	s.Use(helmet.New())
}

func addRequestIDMiddleware(s fiber.Router) {
	s.Use(requestid.New())
}
