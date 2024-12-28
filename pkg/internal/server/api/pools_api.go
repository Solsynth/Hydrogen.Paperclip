package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/server/exts"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

func listPool(c *fiber.Ctx) error {
	pools, err := services.ListAttachmentPool()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(pools)
}

func getPool(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id")
	pool, err := services.GetAttachmentPool(uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(pool)
}

func createPool(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	var data struct {
		Alias       string                      `json:"alias" validate:"required"`
		Name        string                      `json:"name" validate:"required"`
		Description string                      `json:"description"`
		Config      models.AttachmentPoolConfig `json:"config"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	pool := models.AttachmentPool{
		Alias:       data.Alias,
		Name:        data.Name,
		Description: data.Description,
		Config:      datatypes.NewJSONType(data.Config),
		AccountID:   &user.ID,
	}

	if pool, err := services.NewAttachmentPool(pool); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(pool)
	}
}

func updatePool(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	var data struct {
		Alias       string                      `json:"alias" validate:"required"`
		Name        string                      `json:"name" validate:"required"`
		Description string                      `json:"description"`
		Config      models.AttachmentPoolConfig `json:"config"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	id, _ := c.ParamsInt("id")
	pool, err := services.GetAttachmentPoolWithUser(uint(id), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	pool.Alias = data.Alias
	pool.Name = data.Name
	pool.Description = data.Description
	pool.Config = datatypes.NewJSONType(data.Config)

	if pool, err := services.UpdateAttachmentPool(pool); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(pool)
	}
}

func deletePool(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	id, _ := c.ParamsInt("id")
	pool, err := services.GetAttachmentPoolWithUser(uint(id), user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if pool, err := services.DeleteAttachmentPool(pool); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(pool)
	}
}
