package services

import (
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type destinationMapping struct {
	Index int
	Raw   []byte
}

var (
	destinationsByIndex  = make(map[int]destinationMapping)
	destinationsByRegion = make(map[string]destinationMapping)
)

func BuildDestinationMapping() {
	count := 0
	for idx, value := range viper.GetStringSlice("destinations") {
		var parsed models.BaseDestination
		raw, _ := jsoniter.Marshal(value)
		_ = jsoniter.Unmarshal(raw, &parsed)

		mapping := destinationMapping{
			Index: idx,
			Raw:   raw,
		}

		if len(parsed.Region) > 0 {
			destinationsByIndex[idx] = mapping
			destinationsByRegion[parsed.Region] = mapping
		}

		count++
	}

	log.Info().Int("count", count).Msg("Destinations mapping built")
}
