package internal

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	globalDB *gorm.DB

	DB = orenv("DB", "rssy.db")
	PG = os.Getenv("PG") == "true"

	autoMigrate = os.Getenv("AUTO_MIGRATE") == "true"
)

func init() {
	var dialer gorm.Dialector

	if PG {
		dialer = postgres.Open(DB)
	} else {
		dialer = sqlite.Open(DB)
	}

	db, err := gorm.Open(dialer, &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               &Logger{log: log.Default(), prefix: ""},
	})

	if err != nil {
		log.Fatal(err)
	}

	if autoMigrate {
		err = db.AutoMigrate(&Article{}, &Feed{})
		if err != nil {
			log.Fatal(err)
		}
	}

	globalDB = db
}

type Article struct {
	Uid       string `json:"uid" gorm:"column:uid"`
	Name      string `json:"name" gorm:"column:name"`
	FeedID    int64  `json:"feed_id" gorm:"column:feed_id"`
	Email     string `json:"email" gorm:"column:email"`
	Title     string `json:"title" gorm:"column:title"`
	Link      string `json:"link" gorm:"column:link"`
	Read      bool   `json:"read" gorm:"column:read"`
	Hide      bool   `json:"hide" gorm:"column:hide"`
	Deleted   bool   `json:"deleted" gorm:"column:deleted"`
	CreateAt  int64  `json:"create_at" gorm:"column:create_at"`
	PublishAt int64  `json:"publish_at" gorm:"column:publish_at"`
	Content   string `json:"content" gorm:"column:content"`
}

type Feed struct {
	ID                int64  `json:"id" gorm:"column:id"`
	URL               string `json:"url" gorm:"column:url"`
	Title             string `json:"title" gorm:"column:title"`
	CreateAt          int64  `json:"create_at" gorm:"column:create_at"`
	Priority          int    `json:"priority" gorm:"column:priority"`
	LastFetchedAt     int64  `json:"last_fetched_at" gorm:"column:last_fetched_at"`
	Email             string `json:"email" gorm:"column:email"`
	HideUnread        bool   `json:"hide_unread" gorm:"column:hide_unread"`
	EnableReadability bool   `json:"enable_readability" gorm:"column:enable_readability"`
}

const (
	SceneReadability = "readability"
)

func updateFeed(email, id string, hideUnread, enableReadability bool) error {
	feed := getFeed(id, email)

	if feed.ID == 0 || (feed.HideUnread == hideUnread && feed.EnableReadability == enableReadability) {
		return nil
	}

	err := globalDB.Model(Feed{}).Where("email = ? and id = ?", email, id).
		Updates(map[string]interface{}{
			"hide_unread":        hideUnread,
			"enable_readability": enableReadability,
		}).Error
	if err != nil {
		return fmt.Errorf("could not update feed: %v", err)
	}

	err = globalDB.Model(Article{}).Where("email = ? and feed_id = ?", email, id).Update("hide", hideUnread).Error
	if err != nil {
		return fmt.Errorf("could not update articles: %v", err)
	}

	readabilityCache.Delete(SceneReadability, feed.ID)

	return nil
}

func getReadArticle(uid, email string) (Article, error) {
	article := Article{}

	err := globalDB.Where("uid = ? and email = ?", uid, email).First(&article).Error
	if err != nil {
		return article, fmt.Errorf("could not get article: %v", err)
	}

	err = globalDB.Model(article).Where("uid = ? and email = ?", uid, email).Update("read", true).Error
	if err != nil {
		return article, fmt.Errorf("could not read article: %v", err)
	}

	return article, nil
}

func addFeedAndCreateArticles(feedURL, email string) (int64, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return 0, fmt.Errorf("could not fetch feed: %v", err)
	}

	feedID, err := getSetFeed(feedURL, email, feed.Title, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("could not set feed: %v", err)
	}

	articles := make([]*Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		if !rssItemTimeFilter(item, time.Hour*24*7) {
			continue
		}

		articles = append(articles, &Article{
			Uid:       uuid.New().String(),
			Name:      feed.Title,
			FeedID:    feedID,
			Email:     email,
			Title:     item.Title,
			Link:      item.Link,
			Read:      false,
			Hide:      false,
			Deleted:   false,
			Content:   item.Content,
			PublishAt: item.PublishedParsed.Unix(),
			CreateAt:  time.Now().Unix(),
		})
	}

	if err := globalDB.CreateInBatches(articles, 10).Error; err != nil {
		return feedID, fmt.Errorf("could not create articles: %v", err)
	}

	return feedID, nil
}

func parseFeedAndSaveArticles(fd *Feed) ([]*Article, error) {
	fp := gofeed.NewParser()

	feed, err := fp.ParseURL(fd.URL)
	if err != nil {
		log.Errorf("ticker to get feed error: %v", err)
		feed = &gofeed.Feed{}
	}

	feedFilter := func(item *gofeed.Item) bool {
		if item == nil {
			return false
		}
		if fd.LastFetchedAt == 0 {
			return rssItemTimeFilter(item, time.Hour*24*7)
		}

		if item.PublishedParsed == nil {
			return false
		}

		return item.PublishedParsed.After(time.Unix(fd.LastFetchedAt, 0))
	}

	articles := make([]*Article, 0, len(feed.Items))

	for _, item := range feed.Items {
		if !feedFilter(item) {
			continue
		}

		articles = append(articles, &Article{
			Uid:       uuid.New().String(),
			Name:      feed.Title,
			FeedID:    fd.ID,
			Email:     fd.Email,
			Title:     item.Title,
			Link:      item.Link,
			Read:      false,
			Hide:      fd.HideUnread,
			Deleted:   false,
			Content:   item.Content,
			PublishAt: item.PublishedParsed.Unix(),
		})
	}

	if err := globalDB.CreateInBatches(articles, 10).Error; err != nil {
		log.Errorf("could not create articles: %v", err)
	}

	if err := globalDB.Model(&Feed{ID: fd.ID}).
		Update("last_fetched_at", time.Now().Unix()).Error; err != nil {
		log.Errorf("could not update feed item: %v", err)
	}

	return articles, nil
}

func getSetFeed(url, email, title string, lastFetchedAt int64) (int64, error) {
	feed := &Feed{
		URL:           url,
		Title:         title,
		Email:         email,
		CreateAt:      time.Now().Unix(),
		Priority:      1,
		LastFetchedAt: lastFetchedAt,
	}

	result := globalDB.Where("url = ? and email = ?", url, email).FirstOrCreate(feed)
	if result.Error != nil {
		return 0, result.Error
	}

	return feed.ID, nil
}

func getFeed(id, email string) *Feed {
	var feed Feed

	globalDB.Where("id = ? and email = ?", id, email).First(&feed)

	return &feed
}

func getRecentlyArticles(email string) []Article {
	articles := []Article{}

	err := globalDB.Where("email = ? and read = false and hide = false", email).Order("publish_at desc").Find(&articles).Error
	if err != nil {
		log.Infof("could not get articles: %v", err)
		return nil
	}

	return articles
}

func deleteArticle(uid, email string) error {
	err := globalDB.Where("uid = ? AND email = ?", uid, email).Delete(&Article{}).Error
	if err != nil {
		return fmt.Errorf("could not delete article: %v", err)
	}
	return nil
}

func getFeedArticles(email, feedID string) []Article {
	articles := []Article{}

	err := globalDB.Where("email = ? and feed_id = ? and read = false",
		email, feedID).Order("publish_at desc").Find(&articles).Error
	if err != nil {
		log.Infof("could not get articles: %v", err)
		return nil
	}

	return articles
}

func getFeeds(email string) []Feed {
	feeds := []Feed{}

	err := globalDB.Where("email = ?", email).Order("create_at desc").Find(&feeds).Error
	if err != nil {
		log.Infof("could not get feeds: %v", err)
		return nil
	}

	return feeds
}

func getEmailsFeeds(emails []string) []Feed {
	feeds := []Feed{}

	err := globalDB.Where("email in ?", emails).Order("create_at desc").Find(&feeds).Error
	if err != nil {
		log.Infof("could not get feeds: %v", err)
		return nil
	}

	return feeds
}

func deleteFeed(email, id string) {
	err := globalDB.Where("email = ? AND id = ?", email, id).Delete(&Feed{}).Error
	if err != nil {
		log.Infof("could not delete feed: %v", err)
	}

	err = globalDB.Where("email = ? AND feed_id = ?", email, id).Delete(&Article{}).Error
	if err != nil {
		log.Infof("could not delete article: %v", err)
	}
}

func refreshFeed(email, id string) {
	feed := getFeed(id, email)

	if feed.ID == 0 {
		return
	}

	parseFeedAndSaveArticles(feed)
}

func rssItemTimeFilter(item *gofeed.Item, dur time.Duration) bool {
	if item == nil || item.PublishedParsed == nil {
		return false
	}

	return item.PublishedParsed.Unix() > time.Now().Add(-dur).Unix()
}

func getFeedEnableReadability(feedID int64) bool {
	if value, exists := readabilityCache.Get(SceneReadability, feedID); exists {
		return value.(bool)
	}

	var feed Feed
	err := globalDB.Select("enable_readability").Where("id = ?", feedID).First(&feed).Error
	if err != nil {
		log.Infof("could not get feed: %v", err)
		return false
	}

	readabilityCache.Set(SceneReadability, feedID, feed.EnableReadability)

	return feed.EnableReadability
}
