package api

import (
	"fmt"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/gap"
	"git.solsynth.dev/hypernet/passport/pkg/authkit"
	"github.com/spf13/viper"
	"gorm.io/datatypes"
	"strings"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
)

func listAttachment(c *fiber.Ctx) error {
	take := c.QueryInt("take", 0)
	offset := c.QueryInt("offset", 0)

	if take > 100 {
		take = 100
	}

	tx := database.C

	needQuery := true

	var result = make([]models.Attachment, take)
	var idxList []string

	if len(c.Query("id")) > 0 {
		var pendingQueryId []string
		idx := strings.Split(c.Query("id"), ",")
		for p, raw := range idx {
			idxList = append(idxList, raw)
			if val, ok := services.GetAttachmentCache(raw); ok {
				result[p] = val
			} else {
				pendingQueryId = append(pendingQueryId, raw)
			}
		}
		tx = tx.Where("rid IN ?", pendingQueryId)
		needQuery = len(pendingQueryId) > 0
	} else {
		// Do sort this when doesn't filter by the id
		// Because the sort will mess up the result
		tx = tx.Order("created_at DESC")

		// Do not expose un-public indexable attachments
		prefix := viper.GetString("database.prefix")
		tx = tx.
			Joins(fmt.Sprintf("JOIN %sattachment_pools ON %sattachment_pools.id = %sattachments.pool_id", prefix, prefix, prefix)).
			Where(datatypes.JSONQuery(fmt.Sprintf("%sattachment_pools.config", prefix)).Equals(true, "public_indexable"))
	}

	if len(c.Query("author")) > 0 {
		author, err := authkit.GetUserByName(gap.Nx, c.Query("author"))
		if err == nil {
			tx = tx.Where("attachments.account_id = ?", author.ID)
		}
	}

	if pools := c.Query("pools"); len(pools) > 0 {
		prefix := viper.GetString("database.prefix")
		poolAliases := strings.Split(pools, ",")
		tx = tx.
			Joins(fmt.Sprintf("JOIN %sattachment_pools ON %sattachment_pools.id = %sattachments.pool_id", prefix, prefix, prefix)).
			Where(fmt.Sprintf("%sattachment_pools.alias IN ?", prefix), poolAliases)
	}

	if original := c.QueryBool("original", false); original {
		tx = tx.Where("ref_id IS NULL")
	}

	var count int64
	countTx := tx
	if err := countTx.Model(&models.Attachment{}).Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if needQuery {
		var out []models.Attachment
		if err := tx.Offset(offset).Limit(take).Preload("Account").Find(&out).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if len(idxList) == 0 {
			result = out
		} else {
			for _, item := range out {
				for p, id := range idxList {
					if item.Rid == id {
						result[p] = item
					}
				}
			}
		}
	}

	for _, item := range result {
		services.CacheAttachment(item)
	}

	return c.JSON(fiber.Map{
		"count": count,
		"data":  result,
	})
}
