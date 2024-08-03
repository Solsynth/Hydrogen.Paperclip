package services

import (
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
)

func GetStickerPackWithUser(id, userId uint) (models.StickerPack, error) {
	var pack models.StickerPack
	if err := database.C.Where("id = ? AND account_id = ?", id, userId).First(&pack).Error; err != nil {
		return pack, err
	}
	return pack, nil
}

func ListStickerPackWithStickers(take, offset int) ([]models.StickerPack, error) {
	var packs []models.StickerPack
	if err := database.C.Limit(take).Offset(offset).Preload("Stickers").Preload("Stickers.Attachment").Find(&packs).Error; err != nil {
		return packs, err
	}
	return packs, nil
}

func NewStickerPack(user models.Account, prefix, name, desc string) (models.StickerPack, error) {
	pack := models.StickerPack{
		Prefix:      prefix,
		Name:        name,
		Description: desc,
		AccountID:   user.ID,
	}

	if err := database.C.Save(&pack).Error; err != nil {
		return pack, err
	}
	return pack, nil
}

func UpdateStickerPack(pack models.StickerPack) (models.StickerPack, error) {
	if err := database.C.Save(&pack).Error; err != nil {
		return pack, err
	}
	return pack, nil
}

func DeleteStickerPack(pack models.StickerPack) (models.StickerPack, error) {
	if err := database.C.Delete(&pack).Error; err != nil {
		return pack, err
	}
	return pack, nil
}
