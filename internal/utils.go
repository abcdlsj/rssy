package internal

import "os"

func orenv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}
