package internal

import (
	"os"
	"time"
)

var (
	Port       = os.Getenv("PORT")
	GHClientID = os.Getenv("GH_CLIENT_ID")
	GHSecret   = os.Getenv("GH_SECRET")
	SiteURL    = os.Getenv("SITE_URL")

	TimeFormat = "2006-01-02 15:04:05"

	NotiKey = os.Getenv("NOTI_KEY")

	TimeZone = time.FixedZone("CST", 8*3600)

	// 本地调试模式，跳过OAuth登录
	DebugMode     = os.Getenv("DEBUG_MODE") == "true"
	DebugEmail    = orenv("DEBUG_EMAIL", "github@songjian.li")
)
