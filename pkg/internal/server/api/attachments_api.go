package api

import (
	"fmt"
	"net/url"
	"path/filepath"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/server/exts"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

func openAttachment(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)

	metadata, err := services.GetAttachmentByID(uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	var destMap map[string]any
	if metadata.Destination == models.AttachmentDstTemporary {
		destMap = viper.GetStringMap("destinations.temporary")
	} else {
		destMap = viper.GetStringMap("destinations.permanent")
	}

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		if len(metadata.MimeType) > 0 {
			c.Set(fiber.HeaderContentType, metadata.MimeType)
		}
		return c.SendFile(filepath.Join(destConfigured.Path, metadata.Uuid), false)
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		if len(destConfigured.AccessBaseURL) > 0 {
			return c.Redirect(fmt.Sprintf(
				"%s/%s",
				destConfigured.AccessBaseURL,
				url.QueryEscape(filepath.Join(destConfigured.Path, metadata.Uuid)),
			), fiber.StatusMovedPermanently)
		} else {
			protocol := lo.Ternary(destConfigured.EnableSSL, "https", "http")
			return c.Redirect(fmt.Sprintf(
				"%s://%s.%s/%s",
				protocol,
				destConfigured.Bucket,
				destConfigured.Endpoint,
				url.QueryEscape(filepath.Join(destConfigured.Path, metadata.Uuid)),
			), fiber.StatusMovedPermanently)
		}
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
	}
}

func getAttachmentMeta(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id")

	metadata, err := services.GetAttachmentByID(uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	return c.JSON(metadata)
}

func createAttachment(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	usage := c.FormValue("usage")
	if !lo.Contains(viper.GetStringSlice("accepts_usage"), usage) {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("disallowed usage: %s", usage))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return err
	}

	if err := gap.H.EnsureGrantedPerm(c, "CreateAttachments", file.Size); err != nil {
		return err
	}

	usermeta := make(map[string]any)
	_ = jsoniter.UnmarshalFromString(c.FormValue("metadata"), &usermeta)

	tx := database.C.Begin()

	metadata, err := services.NewAttachmentMetadata(tx, user, file, models.Attachment{
		Usage:       usage,
		Alternative: c.FormValue("alt"),
		MimeType:    c.FormValue("mimetype"),
		Metadata:    usermeta,
		IsMature:    len(c.FormValue("mature")) > 0,
		IsAnalyzed:  false,
		Destination: models.AttachmentDstTemporary,
	})
	if err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := services.UploadFileToTemporary(c, file, metadata); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	tx.Commit()

	metadata.Account = user
	services.PublishAnalyzeTask(metadata)

	return c.JSON(metadata)
}

func updateAttachmentMeta(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)

	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var data struct {
		Alternative string `json:"alt"`
		Usage       string `json:"usage"`
		IsMature    bool   `json:"is_mature"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var attachment models.Attachment
	if err := database.C.Where("id = ? AND account_id = ?", id, user.ID).First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	attachment.Alternative = data.Alternative
	attachment.Usage = data.Usage
	attachment.IsMature = data.IsMature

	if attachment, err := services.UpdateAttachment(attachment); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(attachment)
	}
}

func deleteAttachment(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)

	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	attachment, err := services.GetAttachmentByID(uint(id))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	} else if attachment.AccountID != user.ID {
		return fiber.NewError(fiber.StatusNotFound, "record not created by you")
	}

	if err := services.DeleteAttachment(attachment); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.SendStatus(fiber.StatusOK)
	}
}
