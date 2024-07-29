package services

import (
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sync"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const metadataCacheLimit = 512

var metadataCache sync.Map

func GetAttachmentByID(id uint) (models.Attachment, error) {
	if val, ok := metadataCache.Load(id); ok {
		return val.(models.Attachment), nil
	}

	var attachment models.Attachment
	if err := database.C.Where(models.Attachment{
		BaseModel: models.BaseModel{ID: id},
	}).Preload("Account").First(&attachment).Error; err != nil {
		return attachment, err
	} else {
		MaintainAttachmentCache()
		metadataCache.Store(id, attachment)
	}

	return attachment, nil
}

func GetAttachmentByHash(hash string) (models.Attachment, error) {
	var attachment models.Attachment
	if err := database.C.Where(models.Attachment{
		HashCode: hash,
	}).First(&attachment).Error; err != nil {
		return attachment, err
	}
	return attachment, nil
}

func NewAttachmentMetadata(tx *gorm.DB, user *models.Account, file *multipart.FileHeader, attachment models.Attachment) (models.Attachment, error) {
	attachment.Uuid = uuid.NewString()
	attachment.Size = file.Size
	attachment.Name = file.Filename
	attachment.AccountID = user.ID

	// If the user didn't provide file mimetype manually, we have to detect it
	if len(attachment.MimeType) == 0 {
		if ext := filepath.Ext(attachment.Name); len(ext) > 0 {
			// Detect mimetype by file extensions
			attachment.MimeType = mime.TypeByExtension(ext)
		} else {
			// Detect mimetype by file header
			// This method as a fallback method, because this isn't pretty accurate
			header, err := file.Open()
			if err != nil {
				return attachment, fmt.Errorf("failed to read file header: %v", err)
			}
			defer header.Close()

			fileHeader := make([]byte, 512)
			_, err = header.Read(fileHeader)
			if err != nil {
				return attachment, err
			}
			attachment.MimeType = http.DetectContentType(fileHeader)
		}
	}

	if err := tx.Save(&attachment).Error; err != nil {
		return attachment, fmt.Errorf("failed to save attachment record: %v", err)
	} else {
		MaintainAttachmentCache()
		metadataCache.Store(attachment.ID, attachment)
	}

	return attachment, nil
}

func TryLinkAttachment(tx *gorm.DB, og models.Attachment, hash string) (bool, error) {
	prev, err := GetAttachmentByHash(hash)
	if err != nil {
		return false, err
	}

	prev.RefCount++
	og.RefID = &prev.ID
	og.Uuid = prev.Uuid
	og.Destination = prev.Destination

	if err := tx.Save(&og).Error; err != nil {
		tx.Rollback()
		return true, err
	} else if err = tx.Save(&prev).Error; err != nil {
		tx.Rollback()
		return true, err
	}

	metadataCache.Store(prev.ID, prev)
	metadataCache.Store(og.ID, og)

	return true, nil
}

func UpdateAttachment(item models.Attachment) (models.Attachment, error) {
	if err := database.C.Save(&item).Error; err != nil {
		return item, err
	} else {
		MaintainAttachmentCache()
		metadataCache.Store(item.ID, item)
	}

	return item, nil
}

func DeleteAttachment(item models.Attachment) error {
	dat := item

	tx := database.C.Begin()

	if item.RefID != nil {
		var refTarget models.Attachment
		if err := database.C.Where(models.Attachment{
			BaseModel: models.BaseModel{ID: *item.RefID},
		}).First(&refTarget).Error; err == nil {
			refTarget.RefCount--
			if err := tx.Save(&refTarget).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("unable to update ref count: %v", err)
			}
		}
	}
	if err := database.C.Delete(&item).Error; err != nil {
		tx.Rollback()
		return err
	} else {
		metadataCache.Delete(item.ID)
	}

	tx.Commit()

	if dat.RefCount == 0 {
		PublishDeleteFileTask(dat)
	}

	return nil
}

func MaintainAttachmentCache() {
	var keySet []uint
	metadataCache.Range(func(k any, v any) bool {
		keySet = append(keySet, k.(uint))
		return true
	})
	if len(keySet) > metadataCacheLimit {
		go func() {
			log.Debug().Int("count", len(keySet)).Msg("Cleaning attachment metadata cache...")
			for _, k := range keySet {
				metadataCache.Delete(k)
			}
		}()
	}
}
