package api

import (
	"strconv"
	"strings"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/services"
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
	var idxList []uint

	if len(c.Query("id")) > 0 {
		var pendingQueryId []uint
		idx := strings.Split(c.Query("id"), ",")
		for p, raw := range idx {
			id, err := strconv.Atoi(raw)
			if err != nil {
				continue
			} else {
				idxList = append(idxList, uint(id))
			}
			if val, ok := services.GetAttachmentCache(uint(id)); ok {
				result[p] = val
			} else {
				pendingQueryId = append(pendingQueryId, uint(id))
			}
		}
		tx = tx.Where("id IN ?", pendingQueryId)
		needQuery = len(pendingQueryId) > 0
	} else {
		// Do sort this when doesn't filter by the id
		// Because the sort will mess up the result
		tx = tx.Order("created_at DESC")
	}

	if len(c.Query("author")) > 0 {
		var author models.Account
		if err := database.C.Where("name = ?", c.Query("author")).First(&author).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		} else {
			tx = tx.Where("account_id = ?", author.ID)
		}
	}

	if usage := c.Query("usage"); len(usage) > 0 {
		tx = tx.Where("usage IN ?", strings.Split(usage, " "))
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
					if item.ID == id {
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
