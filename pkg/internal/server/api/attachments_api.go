package api

import (
	"fmt"
	"net/url"
	"path/filepath"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/server/exts"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

func openAttachment(c *fiber.Ctx) error {
	id := c.Params("id")

	metadata, err := services.GetAttachmentByRID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	destMap := viper.GetStringMap(fmt.Sprintf("destinations.%d", metadata.Destination))

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		if len(destConfigured.AccessBaseURL) > 0 && !c.QueryBool("direct", false) {
			// This will drop all query parameters,
			// for not it's okay because the openAttachment api won't take any query parameters
			return c.Redirect(fmt.Sprintf(
				"%s%s?direct=true",
				destConfigured.AccessBaseURL,
				c.Path(),
			), fiber.StatusMovedPermanently)
		}
		if len(metadata.MimeType) > 0 {
			c.Set(fiber.HeaderContentType, metadata.MimeType)
		}
		return c.SendFile(filepath.Join(destConfigured.Path, metadata.Uuid))
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
	id := c.Params("id")

	metadata, err := services.GetAttachmentByRID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}

	return c.JSON(metadata)
}

func updateAttachmentMeta(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)
	user := c.Locals("nex_user").(*sec.UserInfo)

	var data struct {
		Thumbnail   *uint          `json:"thumbnail"`
		Compressed  *uint          `json:"compressed"`
		Alternative string         `json:"alt"`
		Metadata    map[string]any `json:"metadata"`
		IsIndexable bool           `json:"is_indexable"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	var attachment models.Attachment
	if err := database.C.
		Where("id = ? AND account_id = ?", id, user.ID).
		Preload("Thumbnail").
		Preload("Compressed").
		First(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if data.Thumbnail != nil && attachment.ThumbnailID != data.Thumbnail {
		var thumbnail models.Attachment
		if err := database.C.
			Where("id = ? AND account_id = ?", data.Thumbnail, user.ID).
			First(&thumbnail).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable find thumbnail: %v", err))
		}
		if attachment.Thumbnail != nil {
			services.UnsetAttachmentAsThumbnail(*attachment.Thumbnail)
		}
		thumbnail, err := services.SetAttachmentAsThumbnail(thumbnail)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable set thumbnail: %v", err))
		}
		attachment.Thumbnail = &thumbnail
		attachment.ThumbnailID = &thumbnail.ID
	}
	if data.Compressed != nil && attachment.CompressedID != data.Compressed {
		var compressed models.Attachment
		if err := database.C.
			Where("id = ? AND account_id = ?", data.Compressed, user.ID).
			First(&compressed).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable find compressed: %v", err))
		}
		if attachment.Compressed != nil {
			services.UnsetAttachmentAsCompressed(*attachment.Compressed)
		}
		compressed, err := services.SetAttachmentAsCompressed(compressed)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable set compressed: %v", err))
		}
		attachment.Compressed = &compressed
		attachment.CompressedID = &compressed.ID
	}

	attachment.Alternative = data.Alternative
	attachment.Usermeta = data.Metadata
	attachment.IsIndexable = data.IsIndexable

	if attachment, err := services.UpdateAttachment(attachment); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		return c.JSON(attachment)
	}
}

func deleteAttachment(c *fiber.Ctx) error {
	id, _ := c.ParamsInt("id", 0)
	user := c.Locals("nex_user").(*sec.UserInfo)

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
