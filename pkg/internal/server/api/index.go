package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	app.Get("/.well-known/destinations", getDestinations)

	api := app.Group(baseURL).Name("API")
	{
		api.Get("/pools", listPost)
		api.Get("/pools/:id", getPool)
		api.Post("/pools", sec.ValidatorMiddleware, createPool)
		api.Put("/pools/:id", sec.ValidatorMiddleware, updatePool)
		api.Delete("/pools/:id", sec.ValidatorMiddleware, deletePool)

		api.Get("/attachments", listAttachment)
		api.Get("/attachments/:id/meta", getAttachmentMeta)
		api.Get("/attachments/:id", openAttachment)
		api.Post("/attachments", sec.ValidatorMiddleware, createAttachmentDirectly)
		api.Put("/attachments/:id", sec.ValidatorMiddleware, updateAttachmentMeta)
		api.Delete("/attachments/:id", sec.ValidatorMiddleware, deleteAttachment)

		api.Post("/attachments/multipart", sec.ValidatorMiddleware, createAttachmentMultipartPlaceholder)
		api.Post("/attachments/multipart/:file/:chunk", sec.ValidatorMiddleware, uploadAttachmentMultipart)

		api.Get("/stickers/lookup", lookupStickerBatch)
		api.Get("/stickers/lookup/:alias", lookupSticker)
		api.Get("/stickers/packs", listStickerPacks)
		api.Get("/stickers/packs/:packId", getStickerPack)
		api.Post("/stickers/packs", sec.ValidatorMiddleware, createStickerPack)
		api.Put("/stickers/packs/:packId", sec.ValidatorMiddleware, updateStickerPack)
		api.Delete("/stickers/packs/:packId", sec.ValidatorMiddleware, deleteStickerPack)

		api.Get("/stickers", listStickers)
		api.Get("/stickers/:stickerId", getSticker)
		api.Post("/stickers", sec.ValidatorMiddleware, createSticker)
		api.Put("/stickers/:stickerId", sec.ValidatorMiddleware, updateSticker)
		api.Delete("/stickers/:stickerId", sec.ValidatorMiddleware, deleteSticker)
	}
}
