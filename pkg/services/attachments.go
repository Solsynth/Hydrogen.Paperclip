package services

import (
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"git.solsynth.dev/hydrogen/paperclip/pkg/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetAttachmentByID(id uint) (models.Attachment, error) {
	var attachment models.Attachment
	if err := database.C.Where(models.Attachment{
		BaseModel: models.BaseModel{ID: id},
	}).First(&attachment).Error; err != nil {
		return attachment, err
	}
	return attachment, nil
}

func GetAttachmentByUUID(id string) (models.Attachment, error) {
	var attachment models.Attachment
	if err := database.C.Where(models.Attachment{
		Uuid: id,
	}).First(&attachment).Error; err != nil {
		return attachment, err
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

func NewAttachmentMetadata(tx *gorm.DB, user models.Account, file *multipart.FileHeader, attachment models.Attachment) (models.Attachment, bool, error) {
	linked := false
	exists, pickupErr := GetAttachmentByHash(attachment.HashCode)
	if pickupErr == nil {
		linked = true
		attachment = exists
		attachment.ID = 0
		attachment.Uuid = uuid.NewString()
		attachment.AccountID = user.ID
	} else {
		// Upload the new file
		attachment.Uuid = uuid.NewString()
		attachment.Size = file.Size
		attachment.Name = file.Filename
		attachment.AccountID = user.ID

		// If user didn't provide file mimetype manually, we gotta to detect it
		if len(attachment.MimeType) == 0 {
			if ext := filepath.Ext(attachment.Name); len(ext) > 0 {
				// Detect mimetype by file extensions
				attachment.MimeType = mime.TypeByExtension(ext)
			} else {
				// Detect mimetype by file header
				// This method as a fallback method, because this isn't pretty accurate
				header, err := file.Open()
				if err != nil {
					return attachment, false, fmt.Errorf("failed to read file header: %v", err)
				}
				defer header.Close()

				fileHeader := make([]byte, 512)
				_, err = header.Read(fileHeader)
				if err != nil {
					return attachment, false, err
				}
				attachment.MimeType = http.DetectContentType(fileHeader)
			}
		}
	}

	if err := tx.Save(&attachment).Error; err != nil {
		return attachment, linked, fmt.Errorf("failed to save attachment record: %v", err)
	}

	return attachment, linked, nil
}

func DeleteAttachment(item models.Attachment) error {
	var dupeCount int64
	if err := database.C.
		Where(&models.Attachment{HashCode: item.HashCode}).
		Model(&models.Attachment{}).
		Count(&dupeCount).Error; err != nil {
		dupeCount = -1
	}

	if err := database.C.Delete(&item).Error; err != nil {
		return err
	}

	if dupeCount != -1 && dupeCount <= 1 {
		return DeleteFile(item)
	}

	return nil
}
