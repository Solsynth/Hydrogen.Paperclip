package services

import (
	"math/rand"
	"strings"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

func RandString(length int) string {
	builder := strings.Builder{}
	builder.Grow(length)
	for idx, cache, remain := length-1, rand.Int63(), letterIdxMax; idx >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			builder.WriteByte(letterBytes[idx])
			idx--
		}
		cache >>= letterIdxBits
		remain--
	}

	return builder.String()
}
