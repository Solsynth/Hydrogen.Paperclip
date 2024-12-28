package services

import (
	"fmt"
	"strings"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
)

func SetAttachmentAsThumbnail(item models.Attachment) (models.Attachment, error) {
	if !strings.HasPrefix(item.MimeType, "image") {
		return item, fmt.Errorf("thumbnail must be an image")
	}

	item.Type = models.AttachmentTypeThumbnail
	item.UsedCount++
	if err := database.C.Save(&item).Error; err != nil {
		return item, err
	}

	return item, nil
}

func SetAttachmentAsCompressed(item models.Attachment) (models.Attachment, error) {
	item.Type = models.AttachmentTypeCompressed
	item.UsedCount++
	if err := database.C.Save(&item).Error; err != nil {
		return item, err
	}

	return item, nil
}

func UnsetAttachmentAsThumbnail(item models.Attachment) (models.Attachment, error) {
	item.Type = models.AttachmentTypeNormal
	item.UsedCount--
	if err := database.C.Save(&item).Error; err != nil {
		return item, err
	}

	return item, nil
}

func UnsetAttachmentAsCompressed(item models.Attachment) (models.Attachment, error) {
	item.Type = models.AttachmentTypeNormal
	item.UsedCount--
	if err := database.C.Save(&item).Error; err != nil {
		return item, err
	}

	return item, nil
}
