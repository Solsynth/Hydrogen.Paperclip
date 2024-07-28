package services

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/database"
	"git.solsynth.dev/hydrogen/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var fileAnalyzeQueue = make(chan models.Attachment, 256)

func PublishAnalyzeTask(file models.Attachment) {
	fileAnalyzeQueue <- file
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
	if _, err := os.Stat(dst); !os.IsExist(err) {
		return fmt.Errorf("attachment doesn't exists in temporary storage")
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

	if err := database.C.Save(&file).Error; err != nil {
		return fmt.Errorf("unable to save file record: %v", err)
	}

	return nil
}
