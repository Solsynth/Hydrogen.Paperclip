package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	nurl "net/url"
	"path/filepath"
	"time"

	localCache "git.solsynth.dev/hypernet/paperclip/pkg/internal/cache"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/store"
	jsoniter "github.com/json-iterator/go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/samber/lo"
)

type openAttachmentResult struct {
	Attachment models.Attachment        `json:"attachment"`
	Boosts     []models.AttachmentBoost `json:"boost"`
}

func GetAttachmentOpenCacheKey(rid string) any {
	return fmt.Sprintf("attachment-open#%s", rid)
}

func OpenAttachmentByRID(rid string, region ...string) (url string, mimetype string, err error) {
	cacheManager := cache.New[any](localCache.S)
	marshal := marshaler.New(cacheManager)
	contx := context.Background()

	var result *openAttachmentResult
	if val, err := marshal.Get(
		contx,
		GetAttachmentOpenCacheKey(rid),
		new(openAttachmentResult),
	); err == nil {
		result = val.(*openAttachmentResult)
	}

	if result == nil {
		var attachment models.Attachment
		if err = database.C.Where(models.Attachment{
			Rid: rid,
		}).
			Preload("Pool").
			Preload("Thumbnail").
			Preload("Compressed").
			Preload("Boosts").
			First(&attachment).Error; err != nil {
			return
		}

		var boosts []models.AttachmentBoost
		boosts, err = ListBoostByAttachment(attachment.ID)
		if err != nil {
			return
		}

		result = &openAttachmentResult{
			Attachment: attachment,
			Boosts:     boosts,
		}
	}

	if len(result.Attachment.MimeType) > 0 {
		mimetype = result.Attachment.MimeType
	}

	var dest models.BaseDestination
	var rawDest []byte

	if len(region) > 0 {
		if des, ok := DestinationsByRegion[region[0]]; ok {
			for _, boost := range result.Boosts {
				if boost.Destination == des.Index {
					rawDest = des.Raw
					json.Unmarshal(rawDest, &dest)
				}
			}
		}
	}
	if rawDest == nil {
		if len(result.Boosts) > 0 {
			randomIdx := rand.IntN(len(result.Boosts))
			if des, ok := DestinationsByIndex[randomIdx]; ok {
				rawDest = des.Raw
				json.Unmarshal(rawDest, &dest)
			}
		} else {
			if des, ok := DestinationsByIndex[result.Attachment.Destination]; ok {
				rawDest = des.Raw
				json.Unmarshal(rawDest, &dest)
			}
		}
	}

	if rawDest == nil {
		err = fmt.Errorf("no destination found")
		return
	}

	switch dest.Type {
	case models.DestinationTypeLocal:
		var destConfigured models.LocalDestination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		url = "file://" + filepath.Join(destConfigured.Path, result.Attachment.Uuid)
		return
	case models.DestinationTypeS3:
		var destConfigured models.S3Destination
		_ = jsoniter.Unmarshal(rawDest, &destConfigured)
		if destConfigured.EnabledSigned {
			var client *minio.Client
			client, err = minio.New(destConfigured.Endpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(destConfigured.SecretID, destConfigured.SecretKey, ""),
				Secure: destConfigured.EnableSSL,
			})
			if err != nil {
				return
			}

			var uri *nurl.URL
			uri, err = client.PresignedGetObject(context.Background(), destConfigured.Bucket, result.Attachment.Uuid, 60*time.Minute, nil)
			if err != nil {
				return
			}

			url = uri.String()
			return
		}
		if len(destConfigured.AccessBaseURL) > 0 {
			url = fmt.Sprintf(
				"%s/%s",
				destConfigured.AccessBaseURL,
				nurl.QueryEscape(filepath.Join(destConfigured.Path, result.Attachment.Uuid)),
			)
		} else {
			protocol := lo.Ternary(destConfigured.EnableSSL, "https", "http")
			url = fmt.Sprintf(
				"%s://%s.%s/%s",
				protocol,
				destConfigured.Bucket,
				destConfigured.Endpoint,
				nurl.QueryEscape(filepath.Join(destConfigured.Path, result.Attachment.Uuid)),
			)
		}
		return
	default:
		err = fmt.Errorf("invalid destination: unsupported protocol %s", dest.Type)
		return
	}
}

func CacheOpenAttachment(item *openAttachmentResult) {
	if item == nil {
		return
	}

	cacheManager := cache.New[any](localCache.S)
	marshal := marshaler.New(cacheManager)
	contx := context.Background()

	_ = marshal.Set(
		contx,
		GetAttachmentCacheKey(item.Attachment.Rid),
		*item,
		store.WithExpiration(60*time.Minute),
		store.WithTags([]string{"attachment-open", fmt.Sprintf("user#%s", item.Attachment.Rid)}),
	)
}
