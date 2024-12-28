package api

import (
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
)

func listDestination(c *fiber.Ctx) error {
	var destinations []models.LocalDestination
	for _, value := range viper.GetStringSlice("destinations") {
		var parsed models.LocalDestination
		raw, _ := jsoniter.Marshal(value)
		_ = jsoniter.Unmarshal(raw, &parsed)
		destinations = append(destinations, parsed)
	}
	return c.JSON(destinations)
}
