package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
		if err := DeleteFile(task); err != nil {
			log.Error().Err(err).Any("task", task).Msg("A file deletion task failed...")
		}
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
