package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"github.com/spf13/viper"
)

func MergeFileChunks(meta models.AttachmentFragment, arrange []string) (models.Attachment, error) {
	attachment := meta.ToAttachment()

	// Fetch destination from config
	destMap := viper.GetStringMapString("destinations.0")

	var dest models.LocalDestination
	dest.Path = destMap["path"]

	// Create the destination file
	destPath := filepath.Join(dest.Path, meta.Uuid)
	destFile, err := os.Create(destPath)
	if err != nil {
		return attachment, err
	}
	defer destFile.Close()

	// 32KB buffer
	buf := make([]byte, 32*1024)

	// Merge the chunks into the destination file
	for _, chunk := range arrange {
		chunkPath := filepath.Join(dest.Path, fmt.Sprintf("%s.part%s", meta.Uuid, chunk))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return attachment, err
		}

		defer chunkFile.Close() // Ensure the file is closed after reading

		for {
			n, err := chunkFile.Read(buf)
			if err != nil && err != io.EOF {
				return attachment, err
			}
			if n == 0 {
				break
			}

			if _, err := destFile.Write(buf[:n]); err != nil {
				return attachment, err
			}
		}
	}

	// Post-upload tasks
	if err := database.C.Save(&attachment).Error; err != nil {
		return attachment, err
	}

	CacheAttachment(attachment)
	PublishAnalyzeTask(attachment)

	// Clean up: remove chunk files
	go DeleteFragment(meta)
	for _, chunk := range arrange {
		chunkPath := filepath.Join(dest.Path, fmt.Sprintf("%s.part%s", meta.Uuid, chunk))
		if err := os.Remove(chunkPath); err != nil {
			return attachment, err
		}
	}

	// Clean up: remove fragment record
	database.C.Delete(&meta)

	return attachment, nil
}
