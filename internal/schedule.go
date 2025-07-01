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
		emails: []string{"github@songjian.li"},
		tk:     time.NewTicker(30 * time.Minute),
	}

	dailyNotifyJob = &DailyNotifyJob{
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

func init() {
	go func() {
		fetchParseJob.Start()
	}()

	go func() {
		dailyNotifyJob.Start()
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
	hour, minute, err := parseNotifyTime()
	if err != nil {
		log.Errorf("failed to parse notify time, %s", NotifyTime)
	}

	log.Infof("parsed notify time H:M=%d:%d (CST/UTC+8)", hour, minute)
	for range t.tk.C {
		// 确保使用中国时区
		now := time.Now().In(TimeZone)
		today := now.Format("2006-01-02")

		if today == t.lastNotifyDate {
			continue
		}

		// 记录当前时间和目标时间，便于调试
		if now.Minute() == 0 || now.Minute() == 30 {
			log.Infof("Current time: %v (CST/UTC+8), target time: %02d:%02d",
				now.Format(TimeFormat), hour, minute)
		}

		if now.Hour() == hour && now.Minute() >= minute && now.Minute() < minute+10 {
			log.Infof("sending daily notification at %v (CST/UTC+8)", now.Format(TimeFormat))
			scheduleSendDailyNotify()
			t.lastNotifyDate = today
		}
	}
}

func (t *DailyNotifyJob) Stop() {
	t.tk.Stop()
}

func parseNotifyTime() (hour, minute int, err error) {
	parts := strings.Split(NotifyTime, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid notify time format, should be HH:MM, got %s", NotifyTime)
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
