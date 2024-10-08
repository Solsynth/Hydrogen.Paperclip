package api

import "github.com/gofiber/fiber/v2"

func MapAPIs(app *fiber.App, baseURL string) {
	app.Get("/.well-known/destinations", getDestinations)

	api := app.Group(baseURL).Name("API")
	{
		api.Get("/pools", listPost)
		api.Get("/pools/:id", getPool)
		api.Post("/pools", createPool)
		api.Put("/pools/:id", updatePool)
		api.Delete("/pools/:id", deletePool)

		api.Get("/attachments", listAttachment)
		api.Get("/attachments/:id/meta", getAttachmentMeta)
		api.Get("/attachments/:id", openAttachment)
		api.Post("/attachments", createAttachmentDirectly)
		api.Put("/attachments/:id", updateAttachmentMeta)
		api.Delete("/attachments/:id", deleteAttachment)

		api.Post("/attachments/multipart", createAttachmentMultipartPlaceholder)
		api.Post("/attachments/multipart/:file/:chunk", uploadAttachmentMultipart)

		api.Get("/stickers/lookup", lookupStickerBatch)
		api.Get("/stickers/lookup/:alias", lookupSticker)
		api.Get("/stickers/packs", listStickerPacks)
		api.Get("/stickers/packs/:packId", getStickerPack)
		api.Post("/stickers/packs", createStickerPack)
		api.Put("/stickers/packs/:packId", updateStickerPack)
		api.Delete("/stickers/packs/:packId", deleteStickerPack)

		api.Get("/stickers", listStickers)
		api.Get("/stickers/:stickerId", getSticker)
		api.Post("/stickers", createSticker)
		api.Put("/stickers/:stickerId", updateSticker)
		api.Delete("/stickers/:stickerId", deleteSticker)
	}
}
