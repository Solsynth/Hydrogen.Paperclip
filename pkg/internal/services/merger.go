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

func MergeFileChunks(meta models.Attachment, arrange []string) (models.Attachment, error) {
	// Fetch destination from config
	destMap := viper.GetStringMapString("destinations.0")

	var dest models.LocalDestination
	dest.Path = destMap["path"]

	// Create the destination file
	destPath := filepath.Join(dest.Path, meta.Uuid)
	destFile, err := os.Create(destPath)
	if err != nil {
		return meta, err
	}
	defer destFile.Close()

	// 32KB buffer
	buf := make([]byte, 32*1024)

	// Merge the chunks into the destination file
	for _, chunk := range arrange {
		chunkPath := filepath.Join(dest.Path, fmt.Sprintf("%s.part%s", meta.Uuid, chunk))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return meta, err
		}

		defer chunkFile.Close() // Ensure the file is closed after reading

		for {
			n, err := chunkFile.Read(buf)
			if err != nil && err != io.EOF {
				return meta, err
			}
			if n == 0 {
				break
			}

			if _, err := destFile.Write(buf[:n]); err != nil {
				return meta, err
			}
		}
	}

	// Post-upload tasks
	meta.IsUploaded = true
	meta.FileChunks = nil
	if err := database.C.Save(&meta).Error; err != nil {
		return meta, err
	}

	CacheAttachment(meta)
	PublishAnalyzeTask(meta)

	// Clean up: remove chunk files
	for _, chunk := range arrange {
		chunkPath := filepath.Join(dest.Path, fmt.Sprintf("%s.part%s", meta.Uuid, chunk))
		if err := os.Remove(chunkPath); err != nil {
			return meta, err
		}
	}

	return meta, nil
}
