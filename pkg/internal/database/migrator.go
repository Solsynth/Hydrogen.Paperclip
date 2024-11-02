package database

import (
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"gorm.io/gorm"
)

var AutoMaintainRange = []any{
	&models.AttachmentPool{},
	&models.Attachment{},
	&models.StickerPack{},
	&models.Sticker{},
}

func RunMigration(source *gorm.DB) error {
	if err := source.AutoMigrate(
		AutoMaintainRange...,
	); err != nil {
		return err
	}

	return nil
}
