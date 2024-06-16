package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"git.solsynth.dev/hydrogen/paperclip/pkg/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

func DeleteFile(meta models.Attachment) error {
	destMap := viper.GetStringMap("destinations")
	dest, destOk := destMap[meta.Destination]
	if !destOk {
		return fmt.Errorf("invalid destination: destination configuration was not found")
	}

	var destParsed models.BaseDestination
	rawDest, _ := jsoniter.Marshal(dest)
	_ = jsoniter.Unmarshal(rawDest, &destParsed)

	switch destParsed.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return DeleteFileFromLocal(destConfigured, meta)
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return DeleteFileFromS3(destConfigured, meta)
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", destParsed.Type)
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
