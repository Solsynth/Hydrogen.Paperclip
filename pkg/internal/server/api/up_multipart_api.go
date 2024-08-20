package api

import (
	"fmt"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func createAttachmentMultipartPlaceholder(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	var data struct {
		Pool        string         `json:"pool" validate:"required"`
		Size        int64          `json:"size" validate:"required"`
		Hash        string         `json:"hash" validate:"required"`
		Alternative string         `json:"alt"`
		MimeType    string         `json:"mimetype"`
		Metadata    map[string]any `json:"metadata"`
		IsMature    bool           `json:"is_mature"`
	}

	aliasingMap := viper.GetStringMapString("pools.aliases")
	if val, ok := aliasingMap[data.Pool]; ok {
		data.Pool = val
	}

	pool, err := services.GetAttachmentPoolByAlias(data.Pool)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("unable to get attachment pool info: %v", err))
	}

	if err = gap.H.EnsureGrantedPerm(c, "CreateAttachments", data.Size); err != nil {
		return err
	} else if pool.Config.Data().MaxFileSize != nil && *pool.Config.Data().MaxFileSize > data.Size {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("attachment pool %s doesn't allow file larger than %d", pool.Alias, *pool.Config.Data().MaxFileSize))
	}

	metadata, err := services.NewAttachmentPlaceholder(database.C, user, models.Attachment{
		UserHash:    &data.Hash,
		Alternative: data.Alternative,
		MimeType:    data.MimeType,
		Metadata:    data.Metadata,
		IsMature:    data.IsMature,
		IsAnalyzed:  false,
		Destination: models.AttachmentDstTemporary,
		Pool:        &pool,
		PoolID:      &pool.ID,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(metadata)
}

func uploadAttachmentMultipart(c *fiber.Ctx) error {
	if err := gap.H.EnsureAuthenticated(c); err != nil {
		return err
	}
	user := c.Locals("user").(models.Account)

	rid := c.Params("file")
	cid := c.Params("chunk")

	file, err := c.FormFile("file")
	if err != nil {
		return err
	} else if file.Size > viper.GetInt64("performance.file_chunk_size") {
		return fiber.NewError(fiber.StatusBadRequest, "file is too large for one chunk")
	}

	meta, err := services.GetAttachmentByRID(rid)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("attachment was not found: %v", err))
	} else if user.ID != meta.AccountID {
		return fiber.NewError(fiber.StatusForbidden, "you are not authorized to upload this attachment")
	}

	if _, ok := meta.FileChunks[cid]; !ok {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("chunk %s was not found", cid))
	} else if services.CheckChunkExistsInTemporary(meta, cid) {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("chunk %s was uploaded", cid))
	}

	if err := services.UploadChunkToTemporary(c, cid, file, meta); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	chunkArrange := make([]string, len(meta.FileChunks))
	isAllUploaded := true
	for cid, idx := range meta.FileChunks {
		if !services.CheckChunkExistsInTemporary(meta, cid) {
			isAllUploaded = false
			break
		} else if val, ok := idx.(int); ok {
			chunkArrange[val] = cid
		}
	}

	if !isAllUploaded {
		database.C.Save(&meta)
		return c.JSON(meta)
	}

	if meta, err = services.MergeFileChunks(meta, chunkArrange); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else {
		return c.JSON(meta)
	}
}
