package backup

import (
	"errors"
	"os"
	"strings"
)

var ErrInvalidLocation = errors.New("location: bucket and key are required")

type Location struct {
	Bucket string
	Key    string
}

var RequiredEnv = []string{"AWS_DEFAULT_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_BUCKET_NAME"}

func LoadEnv() map[string]string {
	env := map[string]string{}
	for _, raw := range os.Environ() {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || !strings.HasPrefix(key, "AWS_") {
			continue
		}

		env[key] = value
	}

	for _, key := range RequiredEnv {
		_, ok := env[key]
		if !ok {
			return nil
		}
	}

	return env
}
