package services

import (
	"fmt"

	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/fs"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func GetBoostByID(id uint) (models.AttachmentBoost, error) {
	var boost models.AttachmentBoost
	if err := database.C.
		Where("id = ?", id).
		Preload("Attachment").
		First(&boost).Error; err != nil {
		return boost, err
	}
	return boost, nil
}

func CreateBoost(user *sec.UserInfo, source models.Attachment, destination int) (models.AttachmentBoost, error) {
	boost := models.AttachmentBoost{
		Status:       models.BoostStatusPending,
		Destination:  destination,
		AttachmentID: source.ID,
		Attachment:   source,
		AccountID:    user.ID,
	}

	dests := cast.ToSlice(viper.Get("destinations"))
	if destination >= len(dests) {
		return boost, fmt.Errorf("invalid destination: %d", destination)
	}

	if err := database.C.Create(&boost).Error; err != nil {
		return boost, err
	}

	boost.Attachment = source
	go ActivateBoost(boost)

	return boost, nil
}

func ActivateBoost(boost models.AttachmentBoost) {
	dests := cast.ToSlice(viper.Get("destinations"))
	if boost.Destination >= len(dests) {
		log.Warn().Any("boost", boost).Msg("Unable to activate boost, invalid destination...")
		database.C.Model(&boost).Update("status", models.BoostStatusError)
		return
	}

	if err := ReUploadFile(boost.Attachment, boost.Destination); err != nil {
		log.Warn().Any("boost", boost).Err(err).Msg("Unable to activate boost...")
		database.C.Model(&boost).Update("status", models.BoostStatusError)
		return
	}

	log.Info().Any("boost", boost).Msg("Boost was activated successfully.")
	database.C.Model(&boost).Update("status", models.BoostStatusActive)
}

func UpdateBoostStatus(boost models.AttachmentBoost, status int) (models.AttachmentBoost, error) {
	if status != models.BoostStatusActive && status != models.BoostStatusSuspended {
		return boost, fmt.Errorf("invalid status: %d", status)
	}
	err := database.C.Save(&boost).Error
	return boost, err
}

func DeleteBoost(boost models.AttachmentBoost) error {
	destMap := viper.GetStringMap(fmt.Sprintf("destinations.%d", boost.Destination))

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return fs.DeleteFileFromLocal(destConfigured, boost.Attachment.Uuid)
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return fs.DeleteFileFromS3(destConfigured, boost.Attachment.Uuid)
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
	}
}
