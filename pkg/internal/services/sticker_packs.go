package services

import (
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"gorm.io/gorm"
)

func GetStickerPack(id uint) (models.StickerPack, error) {
	var pack models.StickerPack
	if err := database.C.Where("id = ?", id).First(&pack).Error; err != nil {
		return pack, err
	}
	return pack, nil
}

func GetStickerPackWithUser(id, userId uint) (models.StickerPack, error) {
	var pack models.StickerPack
	if err := database.C.Where("id = ? AND account_id = ?", id, userId).First(&pack).Error; err != nil {
		return pack, err
	}
	return pack, nil
}

func ListStickerPackWithStickers(tx *gorm.DB, take, offset int) ([]models.StickerPack, error) {
	var packs []models.StickerPack
	if err := tx.Limit(take).Offset(offset).Preload("Stickers").Preload("Stickers.Attachment").Find(&packs).Error; err != nil {
		return packs, err
	}
	return packs, nil
}

func NewStickerPack(user sec.UserInfo, prefix, name, desc string) (models.StickerPack, error) {
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
