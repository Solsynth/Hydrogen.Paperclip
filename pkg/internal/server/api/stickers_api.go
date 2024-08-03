package api

import (
	"fmt"
	"strings"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/server/exts"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

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

	var stickers []models.Sticker
	if err := tx.Limit(take).Offset(offset).Preload("Attachment").Find(&stickers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(stickers)
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
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var data struct {
		Alias        string `json:"alias" validate:"required,alphanum,min=2,max=12"`
		Name         string `json:"name" validate:"required"`
		AttachmentID uint   `json:"attachment_id"`
		PackID       uint   `json:"pack_id"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var attachment models.Attachment
	if err := database.C.Where("id = ?", data.AttachmentID).First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find attachment: %v", err))
	} else if !attachment.IsAnalyzed {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be analyzed")
	}

	if strings.SplitN(attachment.MimeType, "/", 2)[0] != "image" {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be an image")
	} else if width, ok := attachment.Metadata["width"]; !ok {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must has width metadata")
	} else if height, ok := attachment.Metadata["height"]; !ok {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must has height metadata")
	} else if fmt.Sprint(width) != fmt.Sprint(28) || fmt.Sprint(height) != fmt.Sprint(28) {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be a 28x28 image")
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
		AttachmentID: data.AttachmentID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(sticker)
}

func updateSticker(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var data struct {
		Alias        string `json:"alias" validate:"required,alphanum,min=2,max=12"`
		Name         string `json:"name" validate:"required"`
		AttachmentID uint   `json:"attachment_id"`
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
	if err := database.C.Where("id = ?", data.AttachmentID).First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find attachment: %v", err))
	} else if !attachment.IsAnalyzed {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be analyzed")
	}

	if strings.SplitN(attachment.MimeType, "/", 2)[0] != "image" {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be an image")
	} else if width, ok := attachment.Metadata["width"].(float64); !ok {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must has width metadata")
	} else if height, ok := attachment.Metadata["height"].(float64); !ok {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must has height metadata")
	} else if width != 28 || height != 28 {
		return fiber.NewError(fiber.StatusBadRequest, "sticker attachment must be a 28x28 image")
	}

	var pack models.StickerPack
	if err := database.C.Where("id = ? AND account_id = ?", data.PackID, user.ID).First(&pack).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to find pack: %v", err))
	}

	sticker.Alias = data.Alias
	sticker.Name = data.Name
	sticker.PackID = data.PackID
	sticker.AttachmentID = data.AttachmentID

	if sticker, err = services.UpdateSticker(sticker); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(sticker)
}

func deleteSticker(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

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
