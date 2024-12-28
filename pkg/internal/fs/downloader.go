package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

func DownloadFileToLocal(meta models.Attachment, dst int) (string, error) {
	destMap := viper.GetStringMap(fmt.Sprintf("destinations.%d", dst))

	var dest models.BaseDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)

		return filepath.Join(destConfigured.Path, meta.Uuid), nil
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)

		client, err := minio.New(destConfigured.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(destConfigured.SecretID, destConfigured.SecretKey, ""),
			Secure: destConfigured.EnableSSL,
		})
		if err != nil {
			return "", fmt.Errorf("unable to configure s3 client: %v", err)
		}

		inDst := filepath.Join(os.TempDir(), meta.Uuid)

		err = client.FGetObject(context.Background(), destConfigured.Bucket, meta.Uuid, inDst, minio.GetObjectOptions{})
		if err != nil {
			return "", fmt.Errorf("unable to upload file to s3: %v", err)
		}

		return inDst, nil
	default:
		return "", fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
	}
}
