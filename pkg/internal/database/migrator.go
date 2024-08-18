package database

import (
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"gorm.io/gorm"
)

var AutoMaintainRange = []any{
	&models.Account{},
	&models.Attachment{},
	&models.AttachmentPool{},
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
