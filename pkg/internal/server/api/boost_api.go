package api

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/server/exts"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func listBoost(c *fiber.Ctx) error {
	attachmentId, _ := c.ParamsInt("attachmentId", 0)

	if boost, err := services.ListBoostByAttachment(uint(attachmentId)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(boost)
	}
}

func getBoost(c *fiber.Ctx) error {
	boostId, _ := c.ParamsInt("boostId", 0)

	if boost, err := services.GetBoostByID(uint(boostId)); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(boost)
	}
}

func createBoost(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	var data struct {
		Attachment  uint `json:"attachment" validate:"required"`
		Destination int  `json:"destination" validate:"required"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var attachment models.Attachment
	if err := database.C.Where("id = ?", data.Attachment).First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if boost, err := services.CreateBoost(user, attachment, data.Destination); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(boost)
	}
}

func updateBoost(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)
	boostId, _ := c.ParamsInt("boostId", 0)

	var data struct {
		Status int `json:"status" validate:"required"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	boost, err := services.GetBoostByID(uint(boostId))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else if boost.AccountID != user.ID {
		return fiber.NewError(fiber.StatusNotFound, "record not created by you")
	}

	if boost, err := services.UpdateBoostStatus(boost, data.Status); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(boost)
	}
}

func deleteBoost(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)
	boostId, _ := c.ParamsInt("boostId", 0)

	boost, err := services.GetBoostByID(uint(boostId))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else if boost.AccountID != user.ID {
		return fiber.NewError(fiber.StatusNotFound, "record not created by you")
	}

	if err := services.DeleteBoost(boost); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}
