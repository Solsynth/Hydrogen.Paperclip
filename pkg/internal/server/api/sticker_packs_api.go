package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/server/exts"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func listStickerPacks(c *fiber.Ctx) error {
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

	var count int64
	if err := database.C.Model(&models.StickerPack{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var packs []models.StickerPack
	if err := tx.Limit(take).Offset(offset).Find(&packs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  packs,
	})
}

func getStickerPack(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("packId", 0)
	pack, err := services.GetStickerPack(uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(pack)
}

func createStickerPack(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(sec.UserInfo)

	var data struct {
		Prefix      string `json:"prefix" validate:"required,alphanum,min=2,max=12"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	pack, err := services.NewStickerPack(user, data.Prefix, data.Name, data.Description)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(pack)
}

func updateStickerPack(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(sec.UserInfo)

	var data struct {
		Prefix      string `json:"prefix" validate:"required,alphanum,min=2,max=12"`
		Name        string `json:"name" validate:"required"`
		Description string `json:"description"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	id, _ := c.ParamsInt("packId", 0)
	pack, err := services.GetStickerPackWithUser(uint(id), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	pack.Prefix = data.Prefix
	pack.Name = data.Name
	pack.Description = data.Description

	if pack, err = services.UpdateStickerPack(pack); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(pack)
}

func deleteStickerPack(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(sec.UserInfo)

	id, _ := c.ParamsInt("packId", 0)
	pack, err := services.GetStickerPackWithUser(uint(id), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if pack, err = services.DeleteStickerPack(pack); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(pack)
}
