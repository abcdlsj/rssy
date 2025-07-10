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
		err = db.AutoMigrate(&Article{}, &Feed{}, &UserPreference{}, &AISummary{})
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
	Highlight         bool   `json:"highlight" gorm:"column:highlight"`
}

type UserPreference struct {
	ID                 int64  `json:"id" gorm:"primaryKey;column:id"`
	Email              string `json:"email" gorm:"column:email;index"`
	CleanupExpiredDays int    `json:"cleanup_expired_days" gorm:"column:cleanup_expired_days;default:30"`
	EnableAutoCleanup  bool   `json:"enable_auto_cleanup" gorm:"column:enable_auto_cleanup;default:false"`
	NotificationTime   string `json:"notification_time" gorm:"column:notification_time;default:'08:00'"`
	EnableNotification bool   `json:"enable_notification" gorm:"column:enable_notification;default:false"`
	AISummaryPrompt    string `json:"ai_summary_prompt" gorm:"column:ai_summary_prompt;type:text"`
	EnableAISummary    bool   `json:"enable_ai_summary" gorm:"column:enable_ai_summary;default:false"`
	AISummaryTime      string `json:"ai_summary_time" gorm:"column:ai_summary_time;default:'22:00'"`
	CreateAt           int64  `json:"create_at" gorm:"column:create_at"`
	UpdateAt           int64  `json:"update_at" gorm:"column:update_at"`
}

type AISummary struct {
	ID           int64  `json:"id" gorm:"primaryKey;column:id"`
	Email        string `json:"email" gorm:"column:email;index"`
	Date         string `json:"date" gorm:"column:date;index"`
	Title        string `json:"title" gorm:"column:title"`
	Summary      string `json:"summary" gorm:"column:summary;type:text"`
	Categories   string `json:"categories" gorm:"column:categories;type:text"`
	ArticleCount int    `json:"article_count" gorm:"column:article_count"`
	CreateAt     int64  `json:"create_at" gorm:"column:create_at"`
	UpdateAt     int64  `json:"update_at" gorm:"column:update_at"`
}

type FeedMetaCache struct {
	EnableReadability bool
	Highlight         bool
	HideUnread        bool
}

const (
	SceneFeedMeta = "feed_meta"
	SceneUserPref = "user_pref"
)

func updateFeed(email, id string, hideUnread, enableReadability, highlight bool) error {
	feed := getFeed(id, email)

	if feed.ID == 0 || (feed.HideUnread == hideUnread &&
		feed.EnableReadability == enableReadability &&
		feed.Highlight == highlight) {
		return nil
	}

	defer GlobalMemoryCache.Delete(SceneFeedMeta, feed.ID)

	err := globalDB.Model(Feed{}).Where("email = ? and id = ?", email, id).
		Updates(map[string]interface{}{
			"hide_unread":        hideUnread,
			"enable_readability": enableReadability,
			"highlight":          highlight,
		}).Error
	if err != nil {
		return fmt.Errorf("could not update feed: %v", err)
	}

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

	// 获取已存在的文章标题，用于去重
	var existingTitles []string
	if err := globalDB.Model(&Article{}).
		Where("feed_id = ? AND email = ?", fd.ID, fd.Email).
		Pluck("title", &existingTitles).Error; err != nil {
		log.Errorf("failed to fetch existing titles: %v", err)
	}

	// 创建标题映射，用于快速查找
	existingTitlesMap := make(map[string]bool)
	for _, title := range existingTitles {
		existingTitlesMap[title] = true
	}

	articles := make([]*Article, 0, len(feed.Items))

	for _, item := range feed.Items {
		if !feedFilter(item) {
			continue
		}

		// 标题去重检查
		if _, exists := existingTitlesMap[item.Title]; exists {
			log.Infof("skipping duplicate article: %s", item.Title)
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
			Deleted:   false,
			Content:   item.Content,
			PublishAt: item.PublishedParsed.Unix(),
			CreateAt:  time.Now().Unix(),
		})
	}

	err = globalDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(articles, 10).Error; err != nil {
			return fmt.Errorf("could not create articles: %v", err)
		}

		if err := tx.Model(&Feed{ID: fd.ID}).Update("last_fetched_at", time.Now().Unix()).Error; err != nil {
			return fmt.Errorf("could not update feed item: %v", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not parse feed and save articles: %v", err)
	}

	return articles, nil
}

func getUserPreference(email string) (*UserPreference, error) {
	// 先从缓存中获取
	if value, exists := GlobalMemoryCache.Get(SceneUserPref, email); exists {
		if pref, ok := value.(*UserPreference); ok {
			return pref, nil
		}
	}

	var pref UserPreference
	err := globalDB.Where("email = ?", email).First(&pref).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			pref = UserPreference{
				Email:              email,
				CleanupExpiredDays: 30,
				EnableAutoCleanup:  false,
				NotificationTime:   "08:00",
				EnableNotification: false,
				AISummaryPrompt:    getDefaultAISummaryPrompt(),
				EnableAISummary:    false,
				AISummaryTime:      "03:00",
				CreateAt:           time.Now().Unix(),
				UpdateAt:           time.Now().Unix(),
			}
			err = globalDB.Create(&pref).Error
			if err != nil {
				return nil, fmt.Errorf("could not create user preference: %v", err)
			}
		} else {
			return nil, fmt.Errorf("could not get user preference: %v", err)
		}
	}

	// 缓存结果
	GlobalMemoryCache.Set(SceneUserPref, email, &pref)
	return &pref, nil
}

func updateUserPreference(email string, pref *UserPreference) error {
	pref.UpdateAt = time.Now().Unix()
	err := globalDB.Where("email = ?", email).Updates(pref).Error
	if err != nil {
		return fmt.Errorf("could not update user preference: %v", err)
	}
	
	// 更新后删除缓存
	GlobalMemoryCache.Delete(SceneUserPref, email)
	return nil
}

func cleanupExpiredArticles(email string, days int) (int64, error) {
	expiredTime := time.Now().AddDate(0, 0, -days).Unix()
	result := globalDB.Where("email = ? AND read = false AND publish_at < ?", email, expiredTime).Delete(&Article{})
	if result.Error != nil {
		return 0, fmt.Errorf("could not cleanup expired articles: %v", result.Error)
	}
	return result.RowsAffected, nil
}

func cleanupReadArticles(email string) (int64, error) {
	result := globalDB.Where("email = ? AND read = true", email).Delete(&Article{})
	if result.Error != nil {
		return 0, fmt.Errorf("could not cleanup read articles: %v", result.Error)
	}
	return result.RowsAffected, nil
}

func getDefaultAISummaryPrompt() string {
	return `请分析并总结今天的RSS文章内容，按照以下要求：

1. 整体概述：简要概括今天文章的主要话题和趋势
2. 分类整理：将文章按主题分类（如：技术、科学、商业、社会等）
3. 重点摘要：挑选3-5篇最重要或最有价值的文章进行详细摘要
4. 关键观点：提取今天文章中的关键观点和见解
5. 趋势分析：如果发现某些话题或观点重复出现，请指出

请使用清晰的结构化格式输出，方便阅读和理解。`
}

func getAISummariesForUser(email string, limit int) ([]AISummary, error) {
	var summaries []AISummary
	err := globalDB.Where("email = ?", email).Order("date desc").Limit(limit).Find(&summaries).Error
	if err != nil {
		return nil, fmt.Errorf("could not get AI summaries: %v", err)
	}
	return summaries, nil
}

func createAISummary(email, date, title, summary, categories string, articleCount int) error {
	aiSummary := AISummary{
		Email:        email,
		Date:         date,
		Title:        title,
		Summary:      summary,
		Categories:   categories,
		ArticleCount: articleCount,
		CreateAt:     time.Now().Unix(),
		UpdateAt:     time.Now().Unix(),
	}

	err := globalDB.Where("email = ? AND date = ?", email, date).First(&AISummary{}).Error
	if err == gorm.ErrRecordNotFound {
		err = globalDB.Create(&aiSummary).Error
		if err != nil {
			return fmt.Errorf("could not create AI summary: %v", err)
		}
	} else if err == nil {
		aiSummary.UpdateAt = time.Now().Unix()
		err = globalDB.Where("email = ? AND date = ?", email, date).Updates(&aiSummary).Error
		if err != nil {
			return fmt.Errorf("could not update AI summary: %v", err)
		}
	} else {
		return fmt.Errorf("could not check AI summary: %v", err)
	}

	return nil
}

func getArticlesForAISummary(email string, date time.Time) ([]Article, error) {
	// 确保使用上海时区（CST）
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, TimeZone)
	end := start.Add(24 * time.Hour)

	log.Infof("Getting articles for AI summary: email=%s, date=%s, start=%v, end=%v",
		email, date.Format("2006-01-02"), start.Unix(), end.Unix())

	var articles []Article
	err := globalDB.Where("email = ? AND publish_at >= ? AND publish_at < ? AND deleted = false",
		email, start.Unix(), end.Unix()).Find(&articles).Error
	if err != nil {
		return nil, fmt.Errorf("could not get articles for AI summary: %v", err)
	}

	log.Infof("Found %d articles for AI summary", len(articles))
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

	err := globalDB.Where("email = ? and read = false", email).Order("publish_at desc").Find(&articles).Error
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

func getFeedMetaWithCache(feedID int64) FeedMetaCache {
	if value, exists := GlobalMemoryCache.Get(SceneFeedMeta, feedID); exists {
		return value.(FeedMetaCache)
	}

	var feed Feed
	err := globalDB.Select("enable_readability, highlight, hide_unread").Where("id = ?", feedID).First(&feed).Error
	if err != nil {
		log.Infof("could not get feed: %v", err)
		return FeedMetaCache{}
	}

	meta := FeedMetaCache{
		EnableReadability: feed.EnableReadability,
		Highlight:         feed.Highlight,
		HideUnread:        feed.HideUnread,
	}

	GlobalMemoryCache.Set(SceneFeedMeta, feedID, meta)
	return meta
}

func getYesterdayHighlightedUnreadArticlesForUser(email string) ([]Article, error) {
	// 获取该用户高亮的 feed IDs
	var highlightedFeedIDs []int64
	if err := globalDB.Model(&Feed{}).
		Where("email = ? AND highlight = ?", email, true).
		Pluck("id", &highlightedFeedIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch highlighted feed IDs for user %s: %v", email, err)
	}

	if len(highlightedFeedIDs) == 0 {
		return nil, fmt.Errorf("no highlighted feeds found for user %s", email)
	}

	// 使用正确的时区
	yesterday := time.Now().In(TimeZone).Add(-24 * time.Hour)
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, TimeZone)
	end := start.Add(24 * time.Hour)

	var articles []Article
	err := globalDB.Where("email = ? AND publish_at >= ? AND publish_at < ? AND read = ? AND deleted = ? AND feed_id IN ?",
		email, start.Unix(), end.Unix(), false, false, highlightedFeedIDs).
		Find(&articles).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch articles for user %s: %v", email, err)
	}

	return articles, nil
}

