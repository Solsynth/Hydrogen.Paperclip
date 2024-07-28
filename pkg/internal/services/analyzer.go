package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var fileAnalyzeQueue = make(chan models.Attachment, 256)

func PublishAnalyzeTask(file models.Attachment) {
	fileAnalyzeQueue <- file
}

func StartConsumeAnalyzeTask() {
	for {
		task := <-fileAnalyzeQueue
		start := time.Now()
		if err := AnalyzeAttachment(task); err != nil {
			log.Error().Err(err).Any("task", task).Msg("A file analyze task failed...")
		} else {
			log.Info().Dur("elapsed", time.Since(start)).Any("task", task).Msg("A file analyze task was completed.")
		}
	}
}

func AnalyzeAttachment(file models.Attachment) error {
	if file.Destination != models.AttachmentDstTemporary {
		return fmt.Errorf("attachment isn't in temporary storage, unable to analyze")
	}

	destMap := viper.GetStringMap("destinations.temporary")

	var dest models.LocalDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	dst := filepath.Join(dest.Path, file.Uuid)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return fmt.Errorf("attachment doesn't exists in temporary storage: %v", err)
	}

	if t := strings.SplitN(file.MimeType, "/", 2)[0]; t == "image" {
		// Dealing with image
		reader, err := os.Open(dst)
		if err != nil {
			return fmt.Errorf("unable to open file: %v", err)
		}
		defer reader.Close()
		im, _, err := image.Decode(reader)
		if err != nil {
			return fmt.Errorf("unable to decode file as an image: %v", err)
		}
		width := im.Bounds().Dx()
		height := im.Bounds().Dy()
		ratio := width / height
		file.Metadata = map[string]any{
			"width":  width,
			"height": height,
			"ratio":  ratio,
		}
	}

	if hash, err := HashAttachment(file); err != nil {
		return err
	} else {
		file.HashCode = hash
	}

	tx := database.C.Begin()

	linked, err := TryLinkAttachment(tx, file, file.HashCode)
	if linked && err != nil {
		return fmt.Errorf("unable to link file record: %v", err)
	} else if !linked {
		if err := tx.Save(&file).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("unable to save file record: %v", err)
		}
	}

	if !linked {
		if err := ReUploadFileToPermanent(file); err != nil {
			tx.Rollback()
			return fmt.Errorf("unable to move file to permanet storage: %v", err)
		}
	}

	tx.Commit()

	return nil
}

func HashAttachment(file models.Attachment) (hash string, err error) {
	if file.Destination != models.AttachmentDstTemporary {
		err = fmt.Errorf("attachment isn't in temporary storage, unable to hash")
		return
	}

	destMap := viper.GetStringMap("destinations.temporary")

	var dest models.LocalDestination
	rawDest, _ := jsoniter.Marshal(destMap)
	_ = jsoniter.Unmarshal(rawDest, &dest)

	dst := filepath.Join(dest.Path, file.Uuid)
	if _, err = os.Stat(dst); os.IsNotExist(err) {
		err = fmt.Errorf("attachment doesn't exists in temporary storage: %v", err)
		return
	}
	var in *os.File
	in, err = os.Open(dst)
	if err != nil {
		err = fmt.Errorf("unable to open file: %v", err)
		return
	}
	defer in.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, in); err != nil {
		err = fmt.Errorf("unable to hash: %v", err)
		return
	}

	hash = hex.EncodeToString(hasher.Sum(nil))
	return
}
