package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"strings"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/server/exts"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func lookupStickerBatch(c *fiber.Ctx) error {
	probe := c.Query("probe")
	if stickers, err := services.GetStickerLikeAlias(probe); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else {
		return c.JSON(stickers)
	}
}

func lookupSticker(c *fiber.Ctx) error {
	alias := c.Params("alias")
	if sticker, err := services.GetStickerWithAlias(alias); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else {
		return c.JSON(sticker)
	}
}

func listStickers(c *fiber.Ctx) error {
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)

	if take > 100 {
		take = 100
	}

	tx := database.C

	if len(c.Query("author")) > 0 {
		var author models.Account
		if err := database.C.Where("name = ?", c.Query("author")).First(&author).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		} else {
			tx = tx.Where("account_id = ?", author.ID)
		}
	}

	if val := c.QueryInt("pack", 0); val > 0 {
		tx = tx.Where("pack_id = ?", val)
	}

	var count int64
	countTx := tx
	if err := countTx.Model(&models.Sticker{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var stickers []models.Sticker
	if err := tx.Limit(take).Offset(offset).Preload("Attachment").Find(&stickers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  stickers,
	})
}

func getSticker(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("stickerId", 0)
	sticker, err := services.GetSticker(uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return c.JSON(sticker)
}

func createSticker(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(sec.UserInfo)

	var data struct {
		Alias        string `json:"alias" validate:"required,alphanum,min=2,max=12"`
		Name         string `json:"name" validate:"required"`
		AttachmentID string `json:"attachment_id"`
		PackID       uint   `json:"pack_id"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var attachment models.Attachment
	if err := database.C.Where("rid = ?", data.AttachmentID).First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find attachment: %v", err))
	} else if !attachment.IsAnalyzed {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be analyzed")
	}

	if strings.SplitN(attachment.MimeType, "/", 2)[0] != "image" {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be an image")
	}

	var pack models.StickerPack
	if err := database.C.Where("id = ? AND account_id = ?", data.PackID, user.ID).First(&pack).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find pack: %v", err))
	}

	sticker, err := services.NewSticker(models.Sticker{
		Alias:        data.Alias,
		Name:         data.Name,
		Attachment:   attachment,
		AccountID:    user.ID,
		PackID:       pack.ID,
		AttachmentID: attachment.ID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(sticker)
}

func updateSticker(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(sec.UserInfo)

	var data struct {
		Alias        string `json:"alias" validate:"required,alphanum,min=2,max=12"`
		Name         string `json:"name" validate:"required"`
		AttachmentID string `json:"attachment_id"`
		PackID       uint   `json:"pack_id"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	id, _ := c.ParamsInt("stickerId", 0)
	sticker, err := services.GetStickerWithUser(uint(id), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	var attachment models.Attachment
	if err := database.C.Where("rid = ?", data.AttachmentID).First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find attachment: %v", err))
	} else if !attachment.IsAnalyzed {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be analyzed")
	}

	if strings.SplitN(attachment.MimeType, "/", 2)[0] != "image" {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be an image")
	}

	var pack models.StickerPack
	if err := database.C.Where("id = ? AND account_id = ?", data.PackID, user.ID).First(&pack).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find pack: %v", err))
	}

	sticker.Alias = data.Alias
	sticker.Name = data.Name
	sticker.PackID = data.PackID
	sticker.AttachmentID = attachment.ID

	if sticker, err = services.UpdateSticker(sticker); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(sticker)
}

func deleteSticker(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(sec.UserInfo)

	id, _ := c.ParamsInt("stickerId", 0)
	sticker, err := services.GetStickerWithUser(uint(id), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	if sticker, err = services.DeleteSticker(sticker); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(sticker)
}
