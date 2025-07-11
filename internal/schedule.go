package internal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var (
	fetchParseJob = FeedParseJob{
		emails: []string{DefaultEmail},
		tk:     time.NewTicker(30 * time.Minute),
	}

	dailyNotifyJob = &DailyNotifyJob{
		tk: time.NewTicker(time.Minute),
	}

	aiSummaryJob = &AISummaryJob{
		tk: time.NewTicker(time.Minute),
	}
)

type FeedParseJob struct {
	tk     *time.Ticker
	emails []string
}

type DailyNotifyJob struct {
	tk             *time.Ticker
	lastNotifyDate string
}

type AISummaryJob struct {
	tk              *time.Ticker
	lastSummaryDate string
}

func init() {
	go func() {
		fetchParseJob.Start()
	}()

	go func() {
		dailyNotifyJob.Start()
	}()

	go func() {
		aiSummaryJob.Start()
	}()
}

func (t *FeedParseJob) Start() {
	log.Infof("start feed parse job")
	for range t.tk.C {
		log.Infof("ticker to get feed, now: %v", time.Now())

		feeds := getEmailsFeeds(t.emails)

		for _, feedItem := range feeds {
			if time.Now().Before(time.Unix(feedItem.LastFetchedAt+3600, 0)) {
				continue
			}

			if feedItem.ID == 0 {
				continue
			}

			parseFeedAndSaveArticles(&feedItem)
		}
	}
}

func (t *FeedParseJob) Stop() {
	t.tk.Stop()
}

func (t *DailyNotifyJob) Start() {
	log.Infof("start daily notify job")
	for range t.tk.C {
		now := time.Now().In(TimeZone)
		today := now.Format("2006-01-02")

		if today == t.lastNotifyDate {
			continue
		}

		// 获取所有启用了通知的用户
		var preferences []UserPreference
		err := globalDB.Where("enable_notification = ?", true).Find(&preferences).Error
		if err != nil {
			log.Errorf("Failed to get users with notification enabled: %v", err)
			continue
		}

		shouldMarkDone := false
		for _, pref := range preferences {
			hour, minute, err := parseTime(pref.NotificationTime)
			if err != nil {
				log.Errorf("Failed to parse notification time for user %s: %v", pref.Email, err)
				continue
			}

			// 检查是否到了通知时间（允许10分钟窗口）
			if now.Hour() == hour && now.Minute() >= minute && now.Minute() < minute+10 {
				log.Infof("Sending daily notification for user %s at %v", pref.Email, now.Format(TimeFormat))
				scheduleSendDailyNotify(pref.Email)
				shouldMarkDone = true
			}
		}

		if shouldMarkDone {
			t.lastNotifyDate = today
		}
	}
}

func (t *DailyNotifyJob) Stop() {
	t.tk.Stop()
}


func (t *AISummaryJob) Start() {
	log.Infof("start AI summary job")
	for range t.tk.C {
		now := time.Now().In(TimeZone)
		today := now.Format("2006-01-02")

		if today == t.lastSummaryDate {
			continue
		}

		// 获取所有启用了AI总结的用户
		var preferences []UserPreference
		err := globalDB.Where("enable_ai_summary = ?", true).Find(&preferences).Error
		if err != nil {
			log.Errorf("Failed to get users with AI summary enabled: %v", err)
			continue
		}

		for _, pref := range preferences {
			hour, minute, err := parseTime(pref.AISummaryTime)
			if err != nil {
				log.Errorf("Failed to parse AI summary time for user %s: %v", pref.Email, err)
				continue
			}

			// 检查是否到了AI总结时间（允许10分钟窗口）
			if now.Hour() == hour && now.Minute() >= minute && now.Minute() < minute+10 {
				log.Infof("Generating AI summary for user %s at %v", pref.Email, now.Format(TimeFormat))

				// 生成前一天的总结（凌晨时段适合总结前一天的内容）
				yesterday := now.AddDate(0, 0, -1)
				err := generateDailyAISummary(pref.Email, yesterday)
				if err != nil {
					log.Errorf("Failed to generate AI summary for user %s: %v", pref.Email, err)
				}
			}
		}

		// 如果有任何用户的AI总结时间在当前时间，标记今天已处理
		shouldMarkDone := false
		for _, pref := range preferences {
			hour, minute, err := parseTime(pref.AISummaryTime)
			if err != nil {
				continue
			}
			if now.Hour() == hour && now.Minute() >= minute && now.Minute() < minute+10 {
				shouldMarkDone = true
				break
			}
		}

		if shouldMarkDone {
			t.lastSummaryDate = today
		}
	}
}

func (t *AISummaryJob) Stop() {
	t.tk.Stop()
}

func parseTime(timeStr string) (hour, minute int, err error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format, should be HH:MM, got %s", timeStr)
	}

	hour, err = strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err = strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid minute: %s", parts[1])
	}

	return hour, minute, nil
}
