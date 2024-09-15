package services

import (
	"fmt"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"github.com/spf13/viper"
)

func GetStickerWithAlias(alias string) (models.Sticker, error) {
	var sticker models.Sticker
	prefix := viper.GetString("database.prefix")
	if err := database.C.
		Joins(fmt.Sprintf("LEFT JOIN %ssticker_packs pk ON pack_id = pk.id", prefix)).
		Where("CONCAT(pk.prefix, alias) = ?", alias).First(&sticker).Error; err != nil {
		return sticker, err
	}
	return sticker, nil
}

func GetSticker(id uint) (models.Sticker, error) {
	var sticker models.Sticker
	if err := database.C.Where("id = ?", id).Preload("Attachment").First(&sticker).Error; err != nil {
		return sticker, err
	}
	return sticker, nil
}

func GetStickerWithUser(id, userId uint) (models.Sticker, error) {
	var sticker models.Sticker
	if err := database.C.Where("id = ? AND account_id = ?", id, userId).First(&sticker).Error; err != nil {
		return sticker, err
	}
	return sticker, nil
}

func NewSticker(sticker models.Sticker) (models.Sticker, error) {
	if err := database.C.Save(&sticker).Error; err != nil {
		return sticker, err
	}
	return sticker, nil
}

func UpdateSticker(sticker models.Sticker) (models.Sticker, error) {
	if err := database.C.Save(&sticker).Error; err != nil {
		return sticker, err
	}
	return sticker, nil
}

func DeleteSticker(sticker models.Sticker) (models.Sticker, error) {
	if err := database.C.Delete(&sticker).Error; err != nil {
		return sticker, err
	}
	return sticker, nil
}
