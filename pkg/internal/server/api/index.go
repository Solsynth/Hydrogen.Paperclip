package api

import "github.com/gofiber/fiber/v2"

func MapAPIs(app *fiber.App, baseURL string) {
	app.Get("/.well-known/destinations", getDestinations)

	api := app.Group(baseURL).Name("API")
	{
		api.Get("/attachments", list)
		api.Get("/attachments/:id/meta", getAttachmentMeta)
		api.Get("/attachments/:id", openAttachment)
		api.Post("/attachments", createAttachment)
		api.Put("/attachments/:id", updateAttachmentMeta)
		api.Delete("/attachments/:id", deleteAttachment)
	}
}
