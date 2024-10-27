package server

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"strings"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/server/api"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var IReader *sec.InternalTokenReader

type App struct {
	app *fiber.App
}

func NewServer() *App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		EnableIPValidation:    true,
		ServerHeader:          "Hydrogen.Paperclip",
		AppName:               "Hydrogen.Paperclip",
		ProxyHeader:           fiber.HeaderXForwardedFor,
		JSONEncoder:           jsoniter.ConfigCompatibleWithStandardLibrary.Marshal,
		JSONDecoder:           jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal,
		BodyLimit:             512 * 1024 * 1024 * 1024, // 512 TiB
		EnablePrintRoutes:     viper.GetBool("debug.print_routes"),
	})

	app.Use(idempotency.New())
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowMethods: strings.Join([]string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodHead,
			fiber.MethodOptions,
			fiber.MethodPut,
			fiber.MethodDelete,
			fiber.MethodPatch,
		}, ","),
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
	}))

	app.Use(logger.New(logger.Config{
		Format: "${status} | ${latency} | ${method} ${path}\n",
		Output: log.Logger,
	}))

	app.Use(sec.ContextMiddleware(IReader))

	api.MapAPIs(app, "/api")

	return &App{app}
}

func (v *App) Listen() {
	if err := v.app.Listen(viper.GetString("bind")); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when starting server...")
	}
}
