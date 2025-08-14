package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"

	"github.com/zachmann/tip/internal/config"
)

var server *fiber.App

var serverConfig = fiber.Config{
	ReadTimeout:    30 * time.Second,
	WriteTimeout:   90 * time.Second,
	IdleTimeout:    150 * time.Second,
	ReadBufferSize: 8192,
	// WriteBufferSize: 4096,
	ErrorHandler: handleError,
	Network:      "tcp",
}

// Init initializes the server
func Init() {
	server = fiber.New(serverConfig)
	addMiddlewares(server)
	addRoutes(server)
}

func addRoutes(s fiber.Router) {
	s.Post("/", handleRemoteIntrospection)
}

func start(s *fiber.App) {
	if !config.Get().Server.TLS.Enabled {
		log.WithField("port", config.Get().Server.Port).Info("TLS is disabled starting http server")
		log.WithError(s.Listen(fmt.Sprintf(":%d", config.Get().Server.Port))).Fatal()
		return // not reached
	}
	// TLS enabled
	if config.Get().Server.TLS.RedirectHTTP && !config.Get().Server.TLS.UseCustomPort {
		httpServer := fiber.New(serverConfig)
		httpServer.All(
			"*", func(ctx *fiber.Ctx) error {
				//goland:noinspection HttpUrlsUsage
				return ctx.Redirect(
					strings.Replace(ctx.Request().URI().String(), "http://", "https://", 1),
					fiber.StatusPermanentRedirect,
				)
			},
		)
		log.Info("TLS and http redirect enabled, starting redirect server on port 80")
		go func() {
			log.WithError(httpServer.Listen(":80")).Fatal()
		}()
	}
	time.Sleep(time.Millisecond) // This is just for a more pretty output with the tls header printed after the http one
	port := 443
	if config.Get().Server.TLS.UseCustomPort {
		port = config.Get().Server.Port
	}
	log.Infof("TLS enabled, starting https server on port %d", port)
	log.WithError(
		s.ListenTLS(
			fmt.Sprintf(":%d", port),
			config.Get().Server.TLS.Cert,
			config.Get().Server.TLS.Key,
		),
	).Fatal()
}

// Start starts the server
func Start() {
	start(server)
}
