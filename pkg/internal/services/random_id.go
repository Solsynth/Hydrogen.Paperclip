package services

import (
	"math/rand"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandString(length int) string {
	builder := make([]rune, length)
	for i := range builder {
		builder[i] = letters[rand.Intn(len(letters))]
	}
	return string(builder)
}
