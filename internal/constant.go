package internal

import (
	"log"
	"os"
	"time"
)

var (
	Port   = os.Getenv("PORT")
	SiteURL = os.Getenv("SITE_URL")

	TimeFormat = "2006-01-02 15:04:05"
	TimeZone = time.FixedZone("CST", 8*3600)

	// 本地调试模式，跳过OAuth登录
	DebugMode = os.Getenv("DEBUG_MODE") == "true"
	// 默认邮箱，运行时必须指定
	DefaultEmail = getRequiredEnv("DEFAULT_EMAIL")
)

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required but not set", key)
	}
	return value
}
