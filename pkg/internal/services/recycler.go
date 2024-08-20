package services

import (
	"context"
	"fmt"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"github.com/samber/lo"
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

func RunMarkLifecycleDeletionTask() {
	var pools []models.AttachmentPool
	if err := database.C.Find(&pools).Error; err != nil {
		return
	}

	var pendingPools []models.AttachmentPool
	for _, pool := range pools {
		if pool.Config.Data().ExistLifecycle != nil {
			pendingPools = append(pendingPools, pool)
		}
	}

	for _, pool := range pendingPools {
		lifecycle := time.Now().Add(-time.Duration(*pool.Config.Data().ExistLifecycle) * time.Second)
		tx := database.C.
			Where("pool_id = ?", pool.ID).
			Where("created_at < ?", lifecycle).
			Where("cleaned_at IS NULL").
			Updates(&models.Attachment{CleanedAt: lo.ToPtr(time.Now())})
		log.Info().
			Str("pool", pool.Alias).
			Int64("count", tx.RowsAffected).
			Err(tx.Error).
			Msg("Marking attachments as clean needed due to pool's lifecycle configuration...")
	}
}

func RunMarkMultipartDeletionTask() {
	lifecycle := time.Now().Add(-60 * time.Minute)
	tx := database.C.
		Where("created_at < ?", lifecycle).
		Where("is_uploaded = ?", false).
		Where("cleaned_at IS NULL").
		Updates(&models.Attachment{CleanedAt: lo.ToPtr(time.Now())})
	log.Info().
		Int64("count", tx.RowsAffected).
		Err(tx.Error).
		Msg("Marking attachments as clean needed due to multipart lifecycle...")
}

func RunScheduleDeletionTask() {
	var attachments []models.Attachment
	if err := database.C.Where("cleaned_at IS NOT NULL").Find(&attachments).Error; err != nil {
		return
	}

	for _, attachment := range attachments {
		if attachment.RefID != nil {
			continue
		}
		if err := DeleteFile(attachment); err != nil {
			log.Error().
				Uint("id", attachment.ID).
				Msg("An error occurred when deleting marked clean up attachments...")
		}
	}

	database.C.Where("cleaned_at IS NOT NULL").Delete(&models.Attachment{})
}

func DeleteFile(meta models.Attachment) error {
	if !meta.IsUploaded {
		destMap := viper.GetStringMap("destinations.temporary")
		var dest models.LocalDestination
		rawDest, _ := jsoniter.Marshal(destMap)
		_ = jsoniter.Unmarshal(rawDest, &dest)

		for cid := range meta.FileChunks {
			path := filepath.Join(dest.Path, fmt.Sprintf("%s.%s", meta.Uuid, cid))
			_ = os.Remove(path)
		}

		return nil
	}

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
