package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"github.com/gofiber/fiber/v2"
)

func MapAPIs(app *fiber.App, baseURL string) {
	api := app.Group(baseURL).Name("API")
	{
		api.Get("/destinations", listDestination)

		boost := api.Group("/boosts").Name("Boosts API")
		{
			boost.Get("/", listBoostByUser)
			boost.Get("/:boostId", getBoost)
			boost.Post("/", sec.ValidatorMiddleware, createBoost)
			boost.Post("/:boostId/activate", sec.ValidatorMiddleware, activateBoost)
			boost.Put("/:boostId", sec.ValidatorMiddleware, updateBoost)
		}

		pools := api.Group("/pools").Name("Pools API")
		{
			pools.Get("/", listPool)
			pools.Get("/:id", getPool)
			pools.Post("/", sec.ValidatorMiddleware, createPool)
			pools.Put("/:id", sec.ValidatorMiddleware, updatePool)
			pools.Delete("/:id", sec.ValidatorMiddleware, deletePool)
		}

		attachments := api.Group("/attachments").Name("Attachments API")
		{
			attachments.Get("/:attachmentId/boosts", listBoostByAttachment)

			attachments.Get("/", listAttachment)
			attachments.Get("/:id/meta", getAttachmentMeta)
			attachments.Get("/:id", openAttachment)
			attachments.Post("/", sec.ValidatorMiddleware, createAttachmentDirectly)
			attachments.Put("/:id", sec.ValidatorMiddleware, updateAttachmentMeta)
			attachments.Delete("/:id", sec.ValidatorMiddleware, deleteAttachment)
		}

		fragments := api.Group("/fragments").Name("Fragments API")
		{
			fragments.Post("/", sec.ValidatorMiddleware, createAttachmentFragment)
			fragments.Post("/:file/:chunk", sec.ValidatorMiddleware, uploadFragmentChunk)
		}

		stickers := api.Group("/stickers").Name("Stickers API")
		{
			stickers.Get("/lookup", lookupStickerBatch)
			stickers.Get("/lookup/:alias", lookupSticker)

			stickers.Get("/", listStickers)
			stickers.Get("/:stickerId", getSticker)
			stickers.Post("/", sec.ValidatorMiddleware, createSticker)
			stickers.Put("/:stickerId", sec.ValidatorMiddleware, updateSticker)
			stickers.Delete("/:stickerId", sec.ValidatorMiddleware, deleteSticker)

			packs := stickers.Group("/packs").Name("Sticker Packs API")
			{
				packs.Get("/", listStickerPacks)
				packs.Get("/:packId", getStickerPack)
				packs.Post("/", sec.ValidatorMiddleware, createStickerPack)
				packs.Put("/:packId", sec.ValidatorMiddleware, updateStickerPack)
				packs.Delete("/:packId", sec.ValidatorMiddleware, deleteStickerPack)
			}
		}
	}
}
