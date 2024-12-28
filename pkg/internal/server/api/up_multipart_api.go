package api

import (
	"encoding/json"
	"fmt"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/fs"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/server/exts"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func createAttachmentFragment(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	var data struct {
		Pool        string         `json:"pool" validate:"required"`
		Size        int64          `json:"size" validate:"required"`
		FileName    string         `json:"name" validate:"required"`
		Alternative string         `json:"alt"`
		MimeType    string         `json:"mimetype"`
		Fingerprint *string        `json:"fingerprint"`
		Metadata    map[string]any `json:"metadata"`
	}

	if err := exts.BindAndValidate(c, &data); err != nil {
		return err
	}

	aliasingMap := viper.GetStringMapString("pools.aliases")
	if val, ok := aliasingMap[data.Pool]; ok {
		data.Pool = val
	}

	pool, err := services.GetAttachmentPoolByAlias(data.Pool)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to get attachment pool info: %v", err))
	}

	if !user.HasPermNode("CreateAttachments", data.Size) {
		return fiber.NewError(fiber.StatusForbidden, "you are not permitted to create attachments like this large")
	} else if pool.Config.Data().MaxFileSize != nil && *pool.Config.Data().MaxFileSize > data.Size {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("attachment pool %s doesn't allow file larger than %d", pool.Alias, *pool.Config.Data().MaxFileSize))
	}

	metadata, err := services.NewAttachmentFragment(database.C, user, models.AttachmentFragment{
		Name:        data.FileName,
		Size:        data.Size,
		Alternative: data.Alternative,
		MimeType:    data.MimeType,
		Usermeta:    data.Metadata,
		Fingerprint: data.Fingerprint,
		Pool:        &pool,
		PoolID:      &pool.ID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	} else {
		metadata.FileChunksMissing = services.FindFragmentMissingChunks(metadata)
	}

	return c.JSON(fiber.Map{
		"chunk_size":  viper.GetInt64("performance.file_chunk_size"),
		"chunk_count": len(metadata.FileChunks),
		"meta":        metadata,
	})
}

func uploadFragmentChunk(c *fiber.Ctx) error {
	user := c.Locals("nex_user").(*sec.UserInfo)

	rid := c.Params("file")
	cid := c.Params("chunk")

	fileData := c.Body()
	if len(fileData) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no file data")
	} else if len(fileData) > viper.GetInt("performance.file_chunk_size") {
		return fiber.NewError(fiber.StatusBadRequest, "file is too large for one chunk")
	}

	meta, err := services.GetFragmentByRID(rid)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("attachment was not found: %v", err))
	} else if user.ID != meta.AccountID {
		return fiber.NewError(fiber.StatusForbidden, "you are not authorized to upload this attachment")
	}

	if _, ok := meta.FileChunks[cid]; !ok {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("chunk %s was not found", cid))
	} else if services.CheckFragmentChunkExists(meta, cid) {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("chunk %s was uploaded", cid))
	}

	if err := services.UploadFragmentChunkBytes(c, cid, fileData, meta); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	chunkArrange := make([]string, len(meta.FileChunks))
	isAllUploaded := true
	for cid, idx := range meta.FileChunks {
		if !services.CheckFragmentChunkExists(meta, cid) {
			isAllUploaded = false
			break
		} else if val, ok := idx.(json.Number); ok {
			data, _ := val.Int64()
			chunkArrange[data] = cid
		}
	}

	if !isAllUploaded {
		return c.JSON(fiber.Map{
			"is_finished": false,
			"fragment":    meta,
		})
	}

	// Merge & post-upload
	attachment, err := fs.MergeFileChunks(meta, chunkArrange)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Post-upload tasks
	if err := database.C.Save(&attachment).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if !c.QueryBool("analyzeNow", false) {
		services.AnalyzeAttachment(attachment)
	} else {
		services.PublishAnalyzeTask(attachment)
	}

	return c.JSON(fiber.Map{
		"is_finished": true,
		"attachment":  attachment,
	})
}
