package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func getDestinations(c *fiber.Ctx) error {
	var data []string
	for key := range viper.GetStringMap("destinations") {
		data = append(data, key)
	}

	return c.JSON(fiber.Map{
		"data":      data,
		"preferred": viper.GetString("preferred_destination"),
	})
}
