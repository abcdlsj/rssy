package internal

import (
	"os"
)

var (
	Port       = os.Getenv("PORT")
	GHClientID = os.Getenv("GH_CLIENT_ID")
	GHSecret   = os.Getenv("GH_SECRET")
	SiteURL    = os.Getenv("SITE_URL")

	TimeFormat = "2006-01-02 15:04:05"

	NotiKey = os.Getenv("NOTI_KEY")
	// 默认早上 8 点发送
	NotifyTime = orenv("NOTIFY_TIME", "08:00")
)
