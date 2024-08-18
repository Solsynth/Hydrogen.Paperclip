package services

import (
	"context"
	"fmt"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"github.com/samber/lo"
	"gorm.io/gorm/clause"
	"os"
	"path/filepath"
	"time"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var fileDeletionQueue = make(chan models.Attachment, 256)

func PublishDeleteFileTask(file models.Attachment) {
	fileDeletionQueue <- file
}

func StartConsumeDeletionTask() {
	for {
		task := <-fileDeletionQueue
		start := time.Now()
		if err := DeleteFile(task); err != nil {
			log.Error().Err(err).Any("task", task).Msg("A file deletion task failed...")
		} else {
			log.Info().Dur("elapsed", time.Since(start)).Uint("id", task.ID).Msg("A file deletion task was completed.")
		}
	}
}

func RunScheduleDeletionTask() {
	var pools []models.AttachmentPool
	if err := database.C.Find(&pools).Error; err != nil {
		return
	}

	var pendingPools []models.AttachmentPool
	for _, pool := range pendingPools {
		if pool.Config.Data().ExistLifecycle != nil {
			pendingPools = append(pendingPools, pool)
		}
	}

	for _, pool := range pendingPools {
		lifecycle := fmt.Sprintf("%d seconds", *pool.Config.Data().ExistLifecycle)
		var attachments []models.Attachment
		if err := database.C.Where("pool_id = ? AND created_at < NOW() - INTERVAL ?", pool.ID, lifecycle).Find(&attachments).Error; err != nil {
			continue
		}
		log.Info().
			Str("pool", pool.Alias).
			Int("count", len(attachments)).
			Msg("Deleting attachments due to pool's lifecycle configuration...")
		for idx, attachment := range attachments {
			if err := DeleteFile(attachment); err != nil {
				log.Error().
					Str("pool", pool.Alias).
					Uint("id", attachment.ID).
					Msg("An error occurred when deleting attachment due to pool's lifecycle configuration...")
			} else {
				attachments[idx].CleanedAt = lo.ToPtr(time.Now())
			}
		}
		database.C.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).CreateInBatches(attachments, 1000)
	}
}

func DeleteFile(meta models.Attachment) error {
	var destMap map[string]any
	if meta.Destination == models.AttachmentDstTemporary {
		destMap = viper.GetStringMap("destinations.temporary")
	} else {
		destMap = viper.GetStringMap("destinations.permanent")
	}

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return DeleteFileFromLocal(destConfigured, meta)
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return DeleteFileFromS3(destConfigured, meta)
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
	}
}

func DeleteFileFromLocal(config models.LocalDestination, meta models.Attachment) error {
	fullpath := filepath.Join(config.Path, meta.Uuid)
	return os.Remove(fullpath)
}

func DeleteFileFromS3(config models.S3Destination, meta models.Attachment) error {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.SecretID, config.SecretKey, ""),
		Secure: config.EnableSSL,
	})
	if err != nil {
		return fmt.Errorf("unable to configure s3 client: %v", err)
	}

	err = client.RemoveObject(context.Background(), config.Bucket, filepath.Join(config.Path, meta.Uuid), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("unable to upload file to s3: %v", err)
	}

	return nil
}
