package services

import (
	database2 "git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"time"

	"github.com/rs/zerolog/log"
)

func DoAutoDatabaseCleanup() {
	deadline := time.Now().Add(60 * time.Minute)
	log.Debug().Time("deadline", deadline).Msg("Now cleaning up entire database...")

	var count int64
	for _, model := range database2.AutoMaintainRange {
		tx := database2.C.Unscoped().Delete(model, "deleted_at >= ?", deadline)
		if tx.Error != nil {
			log.Error().Err(tx.Error).Msg("An error occurred when running auth context cleanup...")
		}
		count += tx.RowsAffected
	}

	log.Debug().Int64("affected", count).Msg("Clean up entire database accomplished.")
}
