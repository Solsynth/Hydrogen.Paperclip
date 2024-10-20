package services

import (
	"context"
	"fmt"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/spf13/viper"
	"gorm.io/datatypes"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	localCache "git.solsynth.dev/hydrogen/paperclip/pkg/internal/cache"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetAttachmentCacheKey(rid string) any {
	return fmt.Sprintf("attachment#%s", rid)
}

func GetAttachmentByID(id uint) (models.Attachment, error) {
	var attachment models.Attachment
	if err := database.C.
		Where(&hyper.BaseModel{ID: id}).
		Preload("Pool").Preload("Account").
		First(&attachment).Error; err != nil {
		return attachment, err
	} else {
		CacheAttachment(attachment)
	}

	return attachment, nil
}

func GetAttachmentByRID(rid string) (models.Attachment, error) {
	cacheManager := cache.New[any](localCache.S)
	marshal := marshaler.New(cacheManager)
	contx := context.Background()

	if val, err := marshal.Get(
		contx,
		GetAttachmentCacheKey(rid),
		new(models.Attachment),
	); err == nil {
		if val.(models.Attachment).Account.ID > 0 {
			return val.(models.Attachment), nil
		}
	}

	var attachment models.Attachment
	if err := database.C.Where(models.Attachment{
		Rid: rid,
	}).Preload("Pool").Preload("Account").First(&attachment).Error; err != nil {
		return attachment, err
	} else {
		CacheAttachment(attachment)
	}

	return attachment, nil
}

func GetAttachmentByHash(hash string) (models.Attachment, error) {
	var attachment models.Attachment
	if err := database.C.Where(models.Attachment{
		HashCode: hash,
	}).Preload("Pool").First(&attachment).Error; err != nil {
		return attachment, err
	}
	return attachment, nil
}

func GetAttachmentCache(rid string) (models.Attachment, bool) {
	cacheManager := cache.New[any](localCache.S)
	marshal := marshaler.New(cacheManager)
	contx := context.Background()

	if val, err := marshal.Get(
		contx,
		GetAttachmentCacheKey(rid),
		new(models.Attachment),
	); err == nil {
		if val.(models.Attachment).Account.ID > 0 {
			return val.(models.Attachment), true
		}
	}
	return models.Attachment{}, false
}

func CacheAttachment(item models.Attachment) {
	cacheManager := cache.New[any](localCache.S)
	marshal := marshaler.New(cacheManager)
	contx := context.Background()

	marshal.Set(
		contx,
		GetAttachmentCacheKey(item.Rid),
		item,
		store.WithExpiration(60*time.Minute),
		store.WithTags([]string{"attachment", fmt.Sprintf("user#%d", item.AccountID)}),
	)
}

func NewAttachmentMetadata(tx *gorm.DB, user models.Account, file *multipart.FileHeader, attachment models.Attachment) (models.Attachment, error) {
	attachment.Uuid = uuid.NewString()
	attachment.Rid = RandString(16)
	attachment.Size = file.Size
	attachment.Name = file.Filename
	attachment.AccountID = user.ID

	// If the user didn't provide file mimetype manually, we have to detect it
	if len(attachment.MimeType) == 0 {
		if ext := filepath.Ext(attachment.Name); len(ext) > 0 {
			// Detect mimetype by file extensions
			attachment.MimeType = mime.TypeByExtension(ext)
		} else {
			// Detect mimetype by file header
			// This method as a fallback method, because this isn't pretty accurate
			header, err := file.Open()
			if err != nil {
				return attachment, fmt.Errorf("failed to read file header: %v", err)
			}
			defer header.Close()

			fileHeader := make([]byte, 512)
			_, err = header.Read(fileHeader)
			if err != nil {
				return attachment, err
			}
			attachment.MimeType = http.DetectContentType(fileHeader)
		}
	}

	if err := tx.Save(&attachment).Error; err != nil {
		return attachment, fmt.Errorf("failed to save attachment record: %v", err)
	} else {
		CacheAttachment(attachment)
	}

	return attachment, nil
}

func NewAttachmentPlaceholder(tx *gorm.DB, user models.Account, attachment models.Attachment) (models.Attachment, error) {
	attachment.Uuid = uuid.NewString()
	attachment.Rid = RandString(16)
	attachment.IsUploaded = false
	attachment.FileChunks = datatypes.JSONMap{}
	attachment.AccountID = user.ID

	chunkSize := viper.GetInt64("performance.file_chunk_size")
	chunkCount := math.Ceil(float64(attachment.Size) / float64(chunkSize))
	for idx := 0; idx < int(chunkCount); idx++ {
		cid := RandString(8)
		attachment.FileChunks[cid] = idx
	}

	// If the user didn't provide file mimetype manually, we have to detect it
	if len(attachment.MimeType) == 0 {
		if ext := filepath.Ext(attachment.Name); len(ext) > 0 {
			// Detect mimetype by file extensions
			attachment.MimeType = mime.TypeByExtension(ext)
		}
	}

	if err := tx.Save(&attachment).Error; err != nil {
		return attachment, fmt.Errorf("failed to save attachment record: %v", err)
	} else {
		CacheAttachment(attachment)
	}

	return attachment, nil
}

func TryLinkAttachment(tx *gorm.DB, og models.Attachment, hash string) (bool, error) {
	prev, err := GetAttachmentByHash(hash)
	if err != nil {
		return false, err
	}

	if prev.PoolID != nil && og.PoolID != nil && prev.PoolID != og.PoolID && prev.Pool != nil && og.Pool != nil {
		if !prev.Pool.Config.Data().AllowCrossPoolEgress || !og.Pool.Config.Data().AllowCrossPoolIngress {
			// Pool config doesn't allow reference
			return false, nil
		}
	}

	prev.RefCount++
	og.RefID = &prev.ID
	og.Uuid = prev.Uuid
	og.Destination = prev.Destination

	if og.AccountID == prev.AccountID {
		og.IsSelfRef = true
	}

	if err := tx.Save(&og).Error; err != nil {
		tx.Rollback()
		return true, err
	} else if err = tx.Save(&prev).Error; err != nil {
		tx.Rollback()
		return true, err
	}

	CacheAttachment(prev)
	CacheAttachment(og)

	return true, nil
}

func UpdateAttachment(item models.Attachment) (models.Attachment, error) {
	if err := database.C.Updates(&item).Error; err != nil {
		return item, err
	} else {
		CacheAttachment(item)
	}

	return item, nil
}

func DeleteAttachment(item models.Attachment) error {
	dat := item

	tx := database.C.Begin()

	if item.RefID != nil {
		var refTarget models.Attachment
		if err := database.C.Where(models.Attachment{
			BaseModel: hyper.BaseModel{ID: *item.RefID},
		}).First(&refTarget).Error; err == nil {
			refTarget.RefCount--
			if err := tx.Save(&refTarget).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("unable to update ref count: %v", err)
			}
		}
	}
	if err := database.C.Delete(&item).Error; err != nil {
		tx.Rollback()
		return err
	} else {
		cacheManager := cache.New[any](localCache.S)
		marshal := marshaler.New(cacheManager)
		contx := context.Background()
		marshal.Delete(contx, GetAttachmentCacheKey(item.Rid))
	}

	tx.Commit()

	if dat.RefCount == 0 {
		PublishDeleteFileTask(dat)
	}

	return nil
}
