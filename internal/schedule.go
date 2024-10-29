package internal

import (
	"time"

	"github.com/charmbracelet/log"
)

var (
	fetchParseJob = FeedParseJob{
		emails: []string{"github@songjian.li"},
		tk:     time.NewTicker(30 * time.Minute),
	}
)

type FeedParseJob struct {
	tk     *time.Ticker
	emails []string
}

func init() {
	go func() {
		fetchParseJob.Start()
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
