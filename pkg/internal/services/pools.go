package services

import (
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
)

func ListAttachmentPool() ([]models.AttachmentPool, error) {
	var pools []models.AttachmentPool
	if err := database.C.Find(&pools).Error; err != nil {
		return pools, err
	}
	return pools, nil
}

func GetAttachmentPool(id uint) (models.AttachmentPool, error) {
	var pool models.AttachmentPool
	if err := database.C.Where("id = ?", id).First(&pool).Error; err != nil {
		return pool, err
	}
	return pool, nil
}

func GetAttachmentPoolWithUser(id uint, userId uint) (models.AttachmentPool, error) {
	var pool models.AttachmentPool
	if err := database.C.Where("id = ? AND account_id = ?", id, userId).First(&pool).Error; err != nil {
		return pool, err
	}
	return pool, nil
}

func NewAttachmentPool(pool models.AttachmentPool) (models.AttachmentPool, error) {
	if err := database.C.Save(&pool).Error; err != nil {
		return pool, err
	}
	return pool, nil
}

func UpdateAttachmentPool(pool models.AttachmentPool) (models.AttachmentPool, error) {
	if err := database.C.Save(&pool).Error; err != nil {
		return pool, err
	}
	return pool, nil
}

func DeleteAttachmentPool(pool models.AttachmentPool) (models.AttachmentPool, error) {
	if err := database.C.Delete(&pool).Error; err != nil {
		return pool, err
	}
	return pool, nil
}
