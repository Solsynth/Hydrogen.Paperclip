package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

func UploadFileToTemporary(ctx *fiber.Ctx, file *multipart.FileHeader, meta models.Attachment) error {
	destMap := viper.GetStringMap("destinations.temporary")

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		return UploadFileToLocal(destConfigured, ctx, file, meta)
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
	}
}

func ReUploadFileToPermanent(meta models.Attachment) error {
	if meta.Destination != models.AttachmentDstTemporary {
		return fmt.Errorf("attachment isn't in temporary storage, unable to process")
	}

	meta.Destination = models.AttachmentDstPermanent

	destMap := viper.GetStringMap("destinations.permanent")

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	prevDestMap := viper.GetStringMap("destinations.temporary")

	// Currently the temporary destination only support the local
	// So we can do this
	var prevDest models.LocalDestination
	prevRawDest, _ := jsoniter.Marshal(prevDestMap)
	_ = jsoniter.Unmarshal(prevRawDest, &prevDest)

	inDst := filepath.Join(prevDest.Path, meta.Uuid)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)

		in, err := os.Open(inDst)
		if err != nil {
			return fmt.Errorf("unable to open file in temporary storage: %v", err)
		}
		defer in.Close()

		out, err := os.Create(filepath.Join(destConfigured.Path, meta.Uuid))
		if err != nil {
			return fmt.Errorf("unable to open dest file: %v", err)
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("unable to copy data to dest file: %v", err)
		}

		database.C.Save(&meta)
		metadataCache.Store(meta.ID, meta)
		return nil
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)

		client, err := minio.New(destConfigured.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(destConfigured.SecretID, destConfigured.SecretKey, ""),
			Secure: destConfigured.EnableSSL,
		})
		if err != nil {
			return fmt.Errorf("unable to configure s3 client: %v", err)
		}

		_, err = client.FPutObject(context.Background(), destConfigured.Bucket, filepath.Join(destConfigured.Path, meta.Uuid), inDst, minio.PutObjectOptions{
			ContentType:          meta.MimeType,
			SendContentMd5:       false,
			DisableContentSha256: true,
		})
		if err != nil {
			return fmt.Errorf("unable to upload file to s3: %v", err)
		}

		database.C.Save(&meta)
		metadataCache.Store(meta.ID, meta)
		return nil
	default:
		return fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
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

	_, err = client.PutObject(context.Background(), config.Bucket, filepath.Join(config.Path, meta.Uuid), buffer, file.Size, minio.PutObjectOptions{
		ContentType:          meta.MimeType,
		SendContentMd5:       false,
		DisableContentSha256: true,
	})
	if err != nil {
		return fmt.Errorf("unable to upload file to s3: %v", err)
	}

	return nil
}
