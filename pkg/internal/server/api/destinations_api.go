package api

import (
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/models"
	"git.solsynth.dev/hypernet/paperclip/pkg/internal/services"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

func listDestination(c *fiber.Ctx) error {
	var destinations []models.BaseDestination
	for _, value := range services.DestinationsByIndex {
		var parsed models.BaseDestination
		raw, _ := jsoniter.Marshal(value)
		_ = jsoniter.Unmarshal(raw, &parsed)
		destinations = append(destinations, parsed)
	}
	return c.JSON(destinations)
}
