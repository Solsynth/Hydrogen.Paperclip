package services

import (
	"fmt"

	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type destinationMapping struct {
	Index int
	Raw   []byte
}

var (
	DestinationsByIndex  = make(map[int]destinationMapping)
	DestinationsByRegion = make(map[string]destinationMapping)
)

func BuildDestinationMapping() {
	count := len(cast.ToSlice(viper.Get("destinations")))
	for idx := 0; idx < count; idx++ {
		fmt.Println(idx)
		destMap := viper.GetStringMap(fmt.Sprintf("destinations.%d", idx))
		var parsed models.BaseDestination
		raw, _ := jsoniter.Marshal(destMap)
		_ = jsoniter.Unmarshal(raw, &parsed)

		mapping := destinationMapping{
			Index: idx,
			Raw:   raw,
		}

		if len(parsed.Region) > 0 {
			DestinationsByIndex[idx] = mapping
			DestinationsByRegion[parsed.Region] = mapping
		}
	}

	log.Info().Int("count", count).Msg("Destinations mapping built")
}
