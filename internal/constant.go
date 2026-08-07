package internal

import (
	"log"
	"os"
	"time"
)

var (
	Port    = os.Getenv("PORT")
	SiteURL = os.Getenv("SITE_URL")

	TimeFormat = "2006-01-02 15:04:05"
	TimeZone   = time.FixedZone("CST", 8*3600)

	// 本地调试模式，跳过OAuth登录
	DebugMode = os.Getenv("DEBUG_MODE") == "true"
	// 默认邮箱，运行时必须指定
	DefaultEmail = getRequiredEnv("DEFAULT_EMAIL")
	// 信任上游反向代理（如 nexo）设置的 X-Auth-User 请求头
	TrustedProxyAuth = os.Getenv("TRUSTED_AUTH_HEADER") == "true"
	// GitHub OAuth 配置，设置后自动开放 GitHub 登录/注册
	GHClientID = os.Getenv("GH_CLIENT_ID")
	GHSecret   = os.Getenv("GH_SECRET")
	// 会话加密密钥是否由环境变量显式提供
	CipherKeySet = os.Getenv("CIPHER_KEY") != ""
	// 阅读代理服务 URL 模板，使用 %s 作为文章链接的占位符
	ReadabilityURLTemplate = os.Getenv("READABILITY_URL_TEMPLATE")
)

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required but not set", key)
	}
	return value
}
