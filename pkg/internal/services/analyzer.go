package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/samber/lo"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/database"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/k0kubun/go-ansi"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
	"gopkg.in/vansante/go-ffprobe.v2"

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
	if err := database.C.
		Where("is_uploaded = ?", true).
		Where("destination = ? OR is_analyzed = ?", models.AttachmentDstTemporary, false).
		Find(&attachments).Error; err != nil {
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

	if len(file.HashCode) == 0 {
		if hash, err := HashAttachment(file); err != nil {
			return err
		} else {
			file.HashCode = hash
		}
	}

	// Do analyze jobs
	if !file.IsAnalyzed || len(file.HashCode) == 0 {
		destMap := viper.GetStringMap("destinations.0")

		var dest models.LocalDestination
		rawDest, _ := jsoniter.Marshal(destMap)
		_ = jsoniter.Unmarshal(rawDest, &dest)

		start = time.Now()

		dst := filepath.Join(dest.Path, file.Uuid)
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			return fmt.Errorf("attachment doesn't exists in temporary storage: %v", err)
		}

		exifWhitelist := []string{
			"Model", "ShutterSpeed", "ISO", "Megapixels", "Aperture",
			"ColorSpace", "ColorTemperature", "ColorTone", "Contrast",
			"ExposureTime", "FNumber", "FocalLength", "Flash", "HDREffect",
			"LensModel",
		}

		switch strings.SplitN(file.MimeType, "/", 2)[0] {
		case "image":
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
				"exif":   map[string]any{},
			}

			// Removing location EXIF data
			et, err := exiftool.NewExiftool()
			if err == nil {
				defer et.Close()
				exif := et.ExtractMetadata(dst)
				for _, data := range exif {
					for k := range data.Fields {
						if strings.HasPrefix(k, "GPS") {
							data.Clear(k)
						} else if lo.Contains(exifWhitelist, k) {
							file.Metadata["exif"].(map[string]any)[k] = data.Fields[k]
						}
					}
				}
				et.WriteMetadata(exif)
			}
		case "video":
			// Dealing with video
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			data, err := ffprobe.ProbeURL(ctx, dst)
			if err != nil {
				return fmt.Errorf("unable to analyze video information: %v", err)
			}

			stream := data.FirstVideoStream()
			ratio := float64(stream.Width) / float64(stream.Height)
			duration, _ := strconv.ParseFloat(stream.Duration, 64)
			file.Metadata = map[string]any{
				"width":       stream.Width,
				"height":      stream.Height,
				"ratio":       ratio,
				"duration":    duration,
				"bit_rate":    stream.BitRate,
				"codec_name":  stream.CodecName,
				"color_range": stream.ColorRange,
				"color_space": stream.ColorSpace,
				"exif":        map[string]any{},
			}

			// Removing location EXIF data
			et, err := exiftool.NewExiftool()
			if err == nil {
				defer et.Close()
				exif := et.ExtractMetadata(dst)
				for _, data := range exif {
					for k := range data.Fields {
						if strings.HasPrefix(k, "GPS") {
							data.Clear(k)
						} else if lo.Contains(exifWhitelist, k) {
							file.Metadata["exif"].(map[string]any)[k] = data.Fields[k]
						}
					}
				}
				et.WriteMetadata(exif)
			}
		}
	}

	tx := database.C.Begin()

	file.IsAnalyzed = true

	linked, err := TryLinkAttachment(tx, file, file.HashCode)
	if linked && err != nil {
		return fmt.Errorf("unable to link file record: %v", err)
	} else if !linked {
		CacheAttachment(file)
		if err := tx.Save(&file).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("unable to save file record: %v", err)
		}
	}

	tx.Commit()

	log.Info().Dur("elapsed", time.Since(start)).Uint("id", file.ID).Msg("A file analyze task was finished, starting uploading...")

	// Move temporary to permanent
	if !linked {
		go func() {
			start = time.Now()
			if err := ReUploadFileToPermanent(file, 1); err != nil {
				log.Warn().Any("file", file).Err(err).Msg("Unable to move file to permanet storage...")
			} else {
				// Recycle the temporary file
				file.Destination = models.AttachmentDstTemporary
				go DeleteFile(file)
				// Finish
				log.Info().Dur("elapsed", time.Since(start)).Uint("id", file.ID).Msg("A file post-analyze upload task was finished.")
			}
		}()
	} else {
		log.Info().Uint("id", file.ID).Msg("File is linked to exists one, skipping uploading...")
	}

	return nil
}

func HashAttachment(file models.Attachment) (hash string, err error) {
	const chunkSize = 32 * 1024

	destMap := viper.GetStringMapString("destinations.0")
	destPath := filepath.Join(destMap["path"], file.Uuid)

	// Check if the file exists
	fileInfo, err := os.Stat(destPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %v", err)
	}

	// Open the file
	inFile, err := os.Open(destPath)
	if err != nil {
		return "", fmt.Errorf("unable to open file: %v", err)
	}
	defer inFile.Close()

	hasher := sha256.New()

	if chunkSize*3 > fileInfo.Size() {
		// If the total size is smaller than three chunks, then hash the whole file
		buf := make([]byte, fileInfo.Size())
		if _, err := inFile.Read(buf); err != nil && err != io.EOF {
			return "", fmt.Errorf("error reading whole file: %v", err)
		}
		hasher.Write(buf)
	} else {
		// Hash the first 32KB
		buf := make([]byte, chunkSize)
		if _, err := inFile.Read(buf); err != nil && err != io.EOF {
			return "", fmt.Errorf("error reading file: %v", err)
		}
		hasher.Write(buf)

		// Hash the middle 32KB
		middleOffset := fileInfo.Size() / 2
		if _, err := inFile.Seek(middleOffset, io.SeekStart); err != nil {
			return "", fmt.Errorf("error seeking to middle: %v", err)
		}
		if _, err := inFile.Read(buf); err != nil && err != io.EOF {
			return "", fmt.Errorf("error reading middle: %v", err)
		}
		hasher.Write(buf)

		// Hash the last 32KB
		endOffset := fileInfo.Size() - chunkSize
		if _, err := inFile.Seek(endOffset, io.SeekStart); err != nil {
			return "", fmt.Errorf("error seeking to end: %v", err)
		}
		if _, err := inFile.Read(buf); err != nil && err != io.EOF {
			return "", fmt.Errorf("error reading end: %v", err)
		}
		hasher.Write(buf)
	}

	// Hash with the file metadata
	hasher.Write([]byte(fmt.Sprintf("%d", file.Size)))

	// Return the combined hash
	hash = hex.EncodeToString(hasher.Sum(nil))
	return
}
