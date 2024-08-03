package services

import (
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
)

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
