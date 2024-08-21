package services

import (
	"fmt"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
)

func MergeFileChunks(meta models.Attachment, arrange []string) (models.Attachment, error) {
	destMap := viper.GetStringMap("destinations.temporary")

	var dest models.LocalDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	destPath := filepath.Join(dest.Path, meta.Uuid)
	destFile, err := os.Create(destPath)
	if err != nil {
		return meta, err
	}
	defer destFile.Close()

	// Merge files
	for _, chunk := range arrange {
		chunkPath := filepath.Join(dest.Path, fmt.Sprintf("%s.part%s", meta.Uuid, chunk))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return meta, err
		}

		_, err = io.Copy(destFile, chunkFile)
		if err != nil {
			_ = chunkFile.Close()
			return meta, err
		}

		_ = chunkFile.Close()
	}

	// Do post-upload tasks
	meta.IsUploaded = true
	meta.FileChunks = nil
	database.C.Save(&meta)

	CacheAttachment(meta)
	PublishAnalyzeTask(meta)

	// Clean up
	for _, chunk := range arrange {
		chunkPath := filepath.Join(dest.Path, fmt.Sprintf("%s.part%s", meta.Uuid, chunk))
		_ = os.Remove(chunkPath)
	}

	return meta, nil
}
