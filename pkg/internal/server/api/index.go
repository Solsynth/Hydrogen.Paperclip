package api

import "github.com/gofiber/fiber/v2"

func MapAPIs(app *fiber.App) {
	app.Get("/.well-known/destinations", getDestinations)

	api := app.Group("/api").Name("API")
	{
		api.Get("/attachments/:id/meta", getAttachmentMeta)
		api.Get("/attachments/:id", openAttachment)
		api.Post("/attachments", createAttachment)
		api.Put("/attachments/:id", updateAttachmentMeta)
		api.Delete("/attachments/:id", deleteAttachment)
	}
}
