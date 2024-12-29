package api

import (
	"fmt"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
)

func createAttachmentDirectly(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	poolAlias := c.FormValue("pool")

	aliasingMap := viper.GetStringMapString("pools.aliases")
	if val, ok := aliasingMap[poolAlias]; ok {
		poolAlias = val
	}

	pool, err := services.GetAttachmentPoolByAlias(poolAlias)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to get attachment pool info: %v", err))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return err
	}

	if !user.HasPermNode("CreateAttachments", file.Size) {
		return fiber.NewError(fiber.StatusForbidden, "you are not permitted to create attachments like this large")
	} else if pool.Config.Data().MaxFileSize != nil && file.Size > *pool.Config.Data().MaxFileSize {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("attachment pool %s doesn't allow file larger than %d", pool.Alias, *pool.Config.Data().MaxFileSize))
	}

	usermeta := make(map[string]any)
	_ = jsoniter.UnmarshalFromString(c.FormValue("metadata"), &usermeta)

	tx := database.C.Begin()

	metadata, err := services.NewAttachmentMetadata(tx, user, file, models.Attachment{
		Alternative: c.FormValue("alt"),
		MimeType:    c.FormValue("mimetype"),
		Usermeta:    usermeta,
		IsAnalyzed:  false,
		Destination: models.AttachmentDstTemporary,
		Pool:        &pool,
		PoolID:      &pool.ID,
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

	metadata.Pool = &pool

	if c.QueryBool("analyzeNow", false) {
		services.AnalyzeAttachment(metadata)
	} else {
		services.PublishAnalyzeTask(metadata)
	}

	return c.JSON(metadata)
}
