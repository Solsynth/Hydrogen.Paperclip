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
	"github.com/k0kubun/go-ansi"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
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
			log.Info().Dur("elapsed", time.Since(start)).Uint("id", task.ID).Msg("A file analyze task was completed.")
		}
	}
}

func ScanUnanalyzedFileFromDatabase() {
	workers := viper.GetInt("workers.files_analyze")

	if workers < 2 {
		log.Warn().Int("val", workers).Int("min", 2).Msg("The file analyzer does not have enough computing power, and the scan of unanalyzed files will not start...")
	}

	var attachments []models.Attachment
	if err := database.C.Where("destination = ? OR is_analyzed = ?", models.AttachmentDstTemporary, false).Find(&attachments).Error; err != nil {
		log.Error().Err(err).Msg("Scan unanalyzed files from database failed...")
		return
	}

	if len(attachments) == 0 {
		return
	}

	go func() {
		var deletionIdSet []uint
		bar := progressbar.NewOptions(len(attachments),
			progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionSetDescription("Analyzing the unanalyzed files..."),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
		for _, task := range attachments {
			if err := AnalyzeAttachment(task); err != nil {
				log.Error().Err(err).Any("task", task).Msg("A background file analyze task failed...")
				deletionIdSet = append(deletionIdSet, task.ID)
			}
			bar.Add(1)
		}
		log.Info().Int("count", len(attachments)).Int("fails", len(deletionIdSet)).Msg("All unanalyzed files has been analyzed!")

		if len(deletionIdSet) > 0 {
			database.C.Delete(&models.Attachment{}, deletionIdSet)
		}
	}()
}

func AnalyzeAttachment(file models.Attachment) error {
	if file.Destination != models.AttachmentDstTemporary {
		return fmt.Errorf("attachment isn't in temporary storage, unable to analyze")
	}

	var start time.Time

	if !file.IsAnalyzed || len(file.HashCode) == 0 {
		destMap := viper.GetStringMap("destinations.temporary")

		var dest models.LocalDestination
		rawDest, _ := jsoniter.Marshal(destMap)
		_ = jsoniter.Unmarshal(rawDest, &dest)

		start = time.Now()

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
			ratio := float64(width) / float64(height)
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
	}

	tx := database.C.Begin()

	file.IsAnalyzed = true

	linked, err := TryLinkAttachment(tx, file, file.HashCode)
	if linked && err != nil {
		return fmt.Errorf("unable to link file record: %v", err)
	} else if !linked {
		metadataCache.Store(file.ID, file)
		if err := tx.Save(&file).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("unable to save file record: %v", err)
		}
	}

	tx.Commit()

	log.Info().Dur("elapsed", time.Since(start)).Uint("id", file.ID).Msg("A file analyze task was finished, starting uploading...")

	start = time.Now()

	// Move temporary to permanet
	if !linked {
		if err := ReUploadFileToPermanent(file); err != nil {
			return fmt.Errorf("unable to move file to permanet storage: %v", err)
		}
	}

	// Recycle the temporary file
	file.Destination = models.AttachmentDstTemporary
	PublishDeleteFileTask(file)

	// Finish
	log.Info().Dur("elapsed", time.Since(start)).Uint("id", file.ID).Bool("linked", linked).Msg("A file post-analyze upload task was finished.")

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
