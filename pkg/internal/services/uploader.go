package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"git.solsynth.dev/hydrogen/paperclip/pkg/models"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

func UploadFile(destName string, ctx *fiber.Ctx, file *multipart.FileHeader, meta models.Attachment) error {
	destMap := viper.GetStringMap("destinations")
	dest, destOk := destMap[destName]
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
		return UploadFileToLocal(destConfigured, ctx, file, meta)
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return UploadFileToS3(destConfigured, file, meta)
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", destParsed.Type)
	}
}

func UploadFileToLocal(config models.LocalDestination, ctx *fiber.Ctx, file *multipart.FileHeader, meta models.Attachment) error {
	return ctx.SaveFile(file, filepath.Join(config.Path, meta.Uuid))
}

func UploadFileToS3(config models.S3Destination, file *multipart.FileHeader, meta models.Attachment) error {
	header, err := file.Open()
	if err != nil {
		return fmt.Errorf("read upload file: %v", err)
	}
	defer header.Close()

	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, header); err != nil {
		return fmt.Errorf("create io reader for upload file: %v", err)
	}

	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.SecretID, config.SecretKey, ""),
		Secure: config.EnableSSL,
	})
	if err != nil {
		return fmt.Errorf("unable to configure s3 client: %v", err)
	}

	_, err = client.PutObject(context.Background(), config.Bucket, filepath.Join(config.Path, meta.Uuid), buffer, -1, minio.PutObjectOptions{
		ContentType: meta.MimeType,
	})
	if err != nil {
		return fmt.Errorf("unable to upload file to s3: %v", err)
	}

	return nil
}
