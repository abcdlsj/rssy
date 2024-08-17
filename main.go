package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/abcdlsj/cr"
	"github.com/charmbracelet/log"
	humanize "github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	"github.com/sashabaranov/go-openai"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	//go:embed tmpl/*.html
	tmplFS embed.FS

	//go:embed assets/*
	assetFs embed.FS

	tmplFuncs = template.FuncMap{
		"truncate": func(content string, length int) string {
			if len(content) <= length {
				return content
			}
			return content[:length]
		},

		"timeformat": func(t int64) string {
			return humanize.Time(time.Unix(t, 0))
		},

		"colortext": func(content string, color string) string {
			return fmt.Sprintf(`<span style="color: %s">%s</span>`, color, content)
		},

		"safeHTML": func(content string) template.HTML {
			return template.HTML(content)
		},

		"displayContentRead": func(content string) bool {
			return len(content) >= 30
		},
	}

	tmpl = template.Must(template.New("").Funcs(tmplFuncs).ParseFS(tmplFS, "tmpl/*.html"))

	port            = os.Getenv("PORT")
	GHClientID      = os.Getenv("GH_CLIENT_ID")
	GHSecret        = os.Getenv("GH_SECRET")
	SiteURL         = os.Getenv("SITE_URL")
	OPENAI_API_KEY  = os.Getenv("OPENAI_API_KEY")
	OPENAI_ENDPOINT = os.Getenv("OPENAI_ENDPOINT")

	openaiClient *openai.Client

	DB         = orenv("DB", "rssy.db")
	PG         = os.Getenv("PG") == "true"
	TimeFormat = "2006-01-02 15:04:05"

	autoMigrate = os.Getenv("AUTO_MIGRATE") == "true"

	GHRedirectURL = fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=user&redirect_uri=%s",
		GHClientID, fmt.Sprintf("%s/login/callback", SiteURL))

	CipherKey = []byte(orenv("CIPHER_KEY", "0b661f0874117724d1e50746c9fe65d9")) // 32

	globalDB *gorm.DB

	fetchParseJob = FeedParseJob{
		emails: []string{"github@songjian.li"},
		tk:     time.NewTicker(30 * time.Minute),
	}

	globalBuzzingFeed          = BuzzingFeed{}
	globalBuzzingFeedUpdatedAt = time.Unix(0, 0)
	buzzingFeedLoadFromRemote  = os.Getenv("BUZZING_REMOTE") == "true"
)

func init() {
	initDB()

	cfg := openai.DefaultConfig(OPENAI_API_KEY)
	cfg.BaseURL = OPENAI_ENDPOINT
	openaiClient = openai.NewClientWithConfig(cfg)

	gin.SetMode(gin.ReleaseMode)
}

func main() {
	r := gin.Default()

	r.SetFuncMap(tmplFuncs)
	r.SetHTMLTemplate(tmpl)

	go func() {
		fetchParseJob.Start()
	}()

	checklogin := func(c *gin.Context) {
		session, err := checkRefreshGHStatus(c.Request)
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		c.Set("email", session.Email)

		c.Next()
	}

	r.GET("/", checklogin, func(c *gin.Context) {
		email := c.GetString("email")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		articles := getRecentlyArticles(email)
		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles": articles,
			"SiteURL":  SiteURL,
			"Headline": "Unreads",
		})
	})

	r.GET("/feed", checklogin, func(c *gin.Context) {
		email := c.GetString("email")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		feeds := getFeeds(email)
		c.HTML(http.StatusOK, "feed.html", gin.H{
			"Feeds":   feeds,
			"SiteURL": SiteURL,
		})
	})

	r.GET("/feed/:id", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		id := c.Param("id")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		feed := getFeed(id, email)

		if feed.ID == 0 {
			c.String(http.StatusNotFound, "feed not found")
			return
		}

		articles := getFeedArticles(email, id)
		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles":        articles,
			"SiteURL":         SiteURL,
			"Headline":        feed.Title,
			"DisplayRefresh":  true,
			"DisplayCheckbox": true,
			"CheckboxValues": map[string]string{
				"hide_unread": strconv.FormatBool(feed.HideUnread),
			},
			"HideCreateBy":  true,
			"FeedID":        id,
			"LastFetchedAt": feed.LastFetchedAt,
		})
	})

	r.POST("/feed/delete/:id", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		id := c.Param("id")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		deleteFeed(email, id)
		c.Redirect(http.StatusFound, "/feed")
	})

	r.POST("/feed/:id/refresh", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		id := c.Param("id")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		refreshFeed(email, id)
		c.Redirect(http.StatusFound, "/feed/"+id)
	})

	r.POST("/feed/:id/update", checklogin, func(c *gin.Context) {
		hide := c.PostForm("hide_unread") == "true"

		email := c.GetString("email")
		id := c.Param("id")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		updateFeed(email, id, hide)
		c.Redirect(http.StatusFound, "/feed/"+id)
	})

	r.POST("/feed/add", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		feedURL := c.PostForm("url")

		log.Infof("email: %s, feedURL: %s", email, feedURL)

		if email == "" || feedURL == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		feedID, err := addFeedAndCreateArticles(feedURL, email)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/feed/%d", feedID))
	})

	r.POST("/feed/import", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		file, _, err := c.Request.FormFile("opml")
		if err != nil {
			c.String(http.StatusBadRequest, "file upload error: %v", err)
			return
		}
		defer file.Close()

		bytes, err := io.ReadAll(file)
		if err != nil {
			c.String(http.StatusInternalServerError, "file read error: %v", err)
			return
		}

		opml, err := parseOPML(bytes)
		if err != nil {
			c.String(http.StatusInternalServerError, "file parse error: %v", err)
			return
		}

		for _, outline := range opml.Body.Outlines {
			if len(outline.Outlines) != 0 {
				for _, subOutline := range outline.Outlines {
					if subOutline.Type != "rss" {
						continue
					}

					getSetFeed(subOutline.XMLURL, email, subOutline.Text, 0)
				}
			}

			if outline.Type == "rss" {
				getSetFeed(outline.XMLURL, email, outline.Text, 0)
			}
		}

		c.Redirect(http.StatusFound, "/feed")
	})

	r.GET("/feed/export", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		var feeds []Feed
		globalDB.Where("email = ?", email).Find(&feeds)

		output, err := exportOPML(feeds)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Header("Content-Disposition", "attachment; filename=feeds.opml")
		c.Data(http.StatusOK, "application/xml", output)
	})

	r.GET("/article/:uid", checklogin, func(c *gin.Context) {
		uid := c.Param("uid")
		email := c.GetString("email")

		article, err := getReadArticle(uid, email)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Redirect(http.StatusSeeOther, article.Link)
	})

	r.GET("/article/:uid/read", checklogin, func(c *gin.Context) {
		uid := c.Param("uid")
		email := c.GetString("email")

		article, err := getReadArticle(uid, email)
		if err != nil {
			c.String(http.StatusNotFound, "Article not found")
			return
		}
		c.HTML(http.StatusOK, "content.html", gin.H{
			"Title":     article.Title,
			"PublishAt": article.PublishAt,
			"Content":   article.Content,
		})
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		favicon, _ := assetFs.ReadFile("assets/favicon.ico")
		c.Data(http.StatusOK, "image/x-icon", favicon)
	})

	r.GET("/login", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, GHRedirectURL)
	})

	r.GET("/login/callback", func(c *gin.Context) {
		code := c.Query("code")

		ak, sk, expiresIn := getGithubAccessToken(code, "")
		if ak == "" {
			c.String(http.StatusInternalServerError, "<html><body><h1>Failed to login</h1></body></html>")
			return
		}

		login, email := getGithubData(ak)
		if login == "" {
			c.String(http.StatusInternalServerError, "<html><body><h1>Failed to login</h1></body></html>")
			return
		}

		session := Session{
			AK:     ak,
			RK:     sk,
			Expire: int(time.Now().Unix()) + expiresIn,
			Email:  email,
		}

		setCookieSession(c.Writer, "s", session)
		c.Redirect(http.StatusSeeOther, "/")
	})

	r.GET("/stream", checklogin, func(c *gin.Context) {
		email := c.GetString("email")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		buzzingFeed := getBuzzingFeedEvery12Hours()

		c.HTML(http.StatusOK, "stream.html", gin.H{
			"SiteURL":       SiteURL,
			"Groups":        buzzingFeed.Groups,
			"LastFetchTime": globalBuzzingFeedUpdatedAt.Unix(),
		})
	})

	log.Infof("Running on %s", SiteURL)
	r.Run(fmt.Sprintf(":%s", port))
}

func updateFeed(email, id string, hideUnread bool) error {
	feed := getFeed(id, email)

	if feed.ID == 0 || feed.HideUnread == hideUnread {
		return nil
	}

	err := globalDB.Model(Feed{}).Where("email = ? and id = ?", email, id).Update("hide_unread", hideUnread).Error
	if err != nil {
		return fmt.Errorf("could not update feed: %v", err)
	}

	err = globalDB.Model(Article{}).Where("email = ? and feed_id = ?", email, id).Update("hide", hideUnread).Error
	if err != nil {
		return fmt.Errorf("could not update articles: %v", err)
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

type Session struct {
	AK     string `json:"ak"`
	RK     string `json:"rk"`
	Expire int    `json:"ak_expire"`
	Email  string `json:"email"`
}

func checkRefreshGHStatus(r *http.Request) (*Session, error) {
	session := getCookieSession(r)
	if session == nil {
		return nil, fmt.Errorf("browser session is nil")
	}

	if time.Now().Unix() > int64(session.Expire) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func getCookieSession(r *http.Request) *Session {
	s, _ := r.Cookie("s")
	if s == nil || s.Value == "" {
		return nil
	}

	session, err := decryptSession(s.Value)
	if err != nil {
		log.Infof("decrypt session error: %v", err)
		return nil
	}

	return &session
}

func setCookieSession(w http.ResponseWriter, name string, session Session) {
	encryptSess, err := encryptSession(session)
	if err != nil {
		log.Infof("encrypt session error: %v", err)
		return
	}

	cookie := http.Cookie{
		Name:   name,
		Value:  encryptSess,
		MaxAge: 24 * 60 * 60 * 7,
		Path:   "/",
	}

	log.Infof("set cookie: %s, session: %+v\n", cookie.String(), session)
	http.SetCookie(w, &cookie)
}

func getGithubAccessToken(code, rk string) (string, string, int) {
	params := map[string]string{"client_id": GHClientID, "client_secret": GHSecret}
	if rk != "" {
		params["refresh_token"] = rk
		params["grant_type"] = "refresh_token"
	} else {
		params["code"] = code
	}

	rbody, _ := json.Marshal(params)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(rbody))
	if err != nil {
		log.Infof("Error: %s\n", err)
		return "", "", 0
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Infof("Error: %s\n", resperr)
		return "", "", 0
	}

	type githubAKResp struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}

	var ghresp githubAKResp

	err = json.NewDecoder(resp.Body).Decode(&ghresp)
	if err != nil {
		log.Infof("Error: %s\n", err)
		return "", "", 0
	}

	return ghresp.AccessToken, ghresp.RefreshToken, ghresp.ExpiresIn
}

func getGithubData(accessToken string) (string, string) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", ""
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", ""
	}

	type githubDataResp struct {
		Login string `json:"login"`
		Email string `json:"email"`
	}

	var ghresp githubDataResp

	err = json.NewDecoder(resp.Body).Decode(&ghresp)
	if err != nil {
		return "", ""
	}

	log.Infof("github data: %+v", ghresp)
	return ghresp.Login, ghresp.Email
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
	ID            int64  `json:"id" gorm:"column:id"`
	URL           string `json:"url" gorm:"column:url"`
	Title         string `json:"title" gorm:"column:title"`
	CreateAt      int64  `json:"create_at" gorm:"column:create_at"`
	Priority      int    `json:"priority" gorm:"column:priority"`
	LastFetchedAt int64  `json:"last_fetched_at" gorm:"column:last_fetched_at"`
	Email         string `json:"email" gorm:"column:email"`
	HideUnread    bool   `json:"hide_unread" gorm:"column:hide_unread"`
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

type Logger struct {
	log    *log.Logger
	prefix string
}

func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *Logger) Info(ctx context.Context, format string, v ...interface{}) {
	l.log.Infof(l.prefix+format, v...)
}

func (l *Logger) Warn(ctx context.Context, format string, v ...interface{}) {
	l.log.Warnf(l.prefix+format, v...)
}

func (l *Logger) Error(ctx context.Context, format string, v ...interface{}) {
	l.log.Errorf(l.prefix+format, v...)
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	l.log.Infof(l.prefix+"%s|rows:%d|error:%v|time:%s", cr.PLCyan(sql), rows, err, time.Since(begin))
}

func initDB() {
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

func getRecentlyArticles(email string) []Article {
	articles := []Article{}

	err := globalDB.Where("email = ? and read = false and hide = false", email).Order("publish_at desc").Find(&articles).Error
	if err != nil {
		log.Infof("could not get articles: %v", err)
		return nil
	}

	return articles
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

func encryptSession(session Session) (string, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("could not marshal: %v", err)
	}

	return encryptData(data)
}

func decryptSession(str string) (Session, error) {
	var session Session

	data, err := decryptStr(str)
	if err != nil {
		return session, fmt.Errorf("could not decrypt: %v", err)
	}

	err = json.Unmarshal(data, &session)
	if err != nil {
		return session, fmt.Errorf("could not unmarshal: %v", err)
	}

	return session, nil
}

func encryptData(data []byte) (string, error) {
	block, err := aes.NewCipher(CipherKey)
	if err != nil {
		return "", fmt.Errorf("could not create new cipher: %v", err)
	}

	cipherText := make([]byte, aes.BlockSize+len(data))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("could not encrypt: %v", err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], data)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptStr(str string) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, fmt.Errorf("could not base64 decode: %v", err)
	}

	block, err := aes.NewCipher(CipherKey)
	if err != nil {
		return nil, fmt.Errorf("could not create new cipher: %v", err)
	}

	if len(cipherText) < aes.BlockSize {
		return nil, fmt.Errorf("invalid ciphertext block size")
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return cipherText, nil
}

func orenv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}

func rssItemTimeFilter(item *gofeed.Item, dur time.Duration) bool {
	if item == nil || item.PublishedParsed == nil {
		return false
	}

	return item.PublishedParsed.Unix() > time.Now().Add(-dur).Unix()
}

type FeedParseJob struct {
	tk     *time.Ticker
	emails []string
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

type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

type Head struct {
	Title        string `xml:"title"`
	DateCreated  string `xml:"dateCreated"`
	DateModified string `xml:"dateModified"`
}

type Body struct {
	Outlines []Outline `xml:"outline"`
}

type Outline struct {
	Text     string    `xml:"text,attr"`
	Type     string    `xml:"type,attr"`
	XMLURL   string    `xml:"xmlUrl,attr"`
	HTMLURL  string    `xml:"htmlUrl,attr,omitempty"`
	Language string    `xml:"language,attr,omitempty"`
	Title    string    `xml:"title,attr,omitempty"`
	Outlines []Outline `xml:"outline"`
}

func parseOPML(data []byte) (*OPML, error) {
	var opml OPML
	if err := xml.Unmarshal(data, &opml); err != nil {
		return nil, err
	}

	return &opml, nil
}

func exportOPML(feeds []Feed) ([]byte, error) {
	outlines := make([]Outline, len(feeds))
	for i, feed := range feeds {
		outlines[i] = Outline{
			Title:  feed.Title,
			Text:   feed.Title,
			Type:   "rss",
			XMLURL: feed.URL,
		}
	}

	opml := OPML{
		Version: "2.0",
		Head: Head{
			Title:        "My Feeds",
			DateCreated:  time.Now().Format(time.RFC1123Z),
			DateModified: time.Now().Format(time.RFC1123Z),
		},
		Body: Body{
			Outlines: outlines,
		},
	}

	bytes, err := xml.MarshalIndent(opml, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func aiCompletion(client *openai.Client, prompt, content string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: prompt},
				{Role: openai.ChatMessageRoleUser, Content: content},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to create completion: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}

type BuzzingFeed struct {
	Version         string `json:"version"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Icon            string `json:"icon"`
	AppleTouchIcon  string `json:"_apple_touch_icon"`
	Favicon         string `json:"favicon"`
	LatestBuildTime string `json:"_latest_build_time"`
	Language        string `json:"language"`
	SiteVersion     string `json:"_site_version"`
	HomePageURL     string `json:"home_page_url"`
	FeedURL         string `json:"feed_url"`
	// Items           []struct {
	// 	Title             string `json:"title"`
	// 	Summary           string `json:"summary"`
	// 	ContentText       string `json:"content_text"`
	// 	ContentHTML       string `json:"content_html"`
	// 	ID                string `json:"id"`
	// 	URL               string `json:"url"`
	// 	DatePublished     string `json:"date_published"`
	// 	DateModified      string `json:"date_modified"`
	// 	OriginalPublished string `json:"_original_published"`
	// 	OriginalLanguage  string `json:"_original_language"`
	// 	Translations      struct {
	// 		En struct {
	// 			Title string `json:"title"`
	// 		} `json:"en"`
	// 		ZhHans struct {
	// 			Title string `json:"title"`
	// 		} `json:"zh-Hans"`
	// 		Ja struct {
	// 			Title string `json:"title"`
	// 		} `json:"ja"`
	// 		ZhHant struct {
	// 			Title string `json:"title"`
	// 		} `json:"zh-Hant"`
	// 	} `json:"_translations"`
	// 	Authors []struct {
	// 		Name string `json:"name"`
	// 		URL  string `json:"url"`
	// 	} `json:"authors,omitempty"`
	// 	Score       int `json:"_score,omitempty"`
	// 	NumComments int `json:"_num_comments,omitempty"`
	// 	Links       []struct {
	// 		URL  string `json:"url"`
	// 		Name string `json:"name"`
	// 	} `json:"_links,omitempty"`
	// 	LiteContentHTML string `json:"_lite_content_html"`
	// 	Author          struct {
	// 		Name string `json:"name"`
	// 		URL  string `json:"url"`
	// 	} `json:"author,omitempty"`
	// 	SiteIdentifier string   `json:"_site_identifier"`
	// 	HumanTime      string   `json:"_human_time"`
	// 	Category       string   `json:"_category"`
	// 	Order          int      `json:"order"`
	// 	Image          string   `json:"image,omitempty"`
	// 	Tags           []string `json:"tags,omitempty"`
	// 	TitlePrefix    string   `json:"_title_prefix,omitempty"`
	// 	TagLinks       []struct {
	// 		Name string `json:"name"`
	// 		URL  string `json:"url"`
	// 	} `json:"_tag_links,omitempty"`
	// 	TitleSuffix string `json:"_title_suffix,omitempty"`
	// 	Video       struct {
	// 		Sources []struct {
	// 			URL string `json:"url"`
	// 		} `json:"sources"`
	// 		Width  int    `json:"width"`
	// 		Height int    `json:"height"`
	// 		Poster string `json:"poster"`
	// 	} `json:"_video,omitempty"`
	// 	Embed struct {
	// 		Provider string `json:"provider"`
	// 		Type     string `json:"type"`
	// 		URL      string `json:"url"`
	// 	} `json:"_embed,omitempty"`
	// 	Sensitive bool `json:"_sensitive,omitempty"`
	// } `json:"items"`
	Sources     []interface{} `json:"_sources"`
	IsLite      bool          `json:"_is_lite"`
	AdviceURL   string        `json:"_advice_url"`
	TitleSuffix string        `json:"_title_suffix"`
	SiteTags    []string      `json:"_site_tags"`
	Groups      []struct {
		Title          string `json:"title"`
		Hostname       string `json:"hostname"`
		SiteIdentifier string `json:"site_identifier"`
		Related        []struct {
			Title      string `json:"title"`
			ShortTitle string `json:"short_title"`
			URL        string `json:"url"`
		} `json:"related"`
		HomePageURL         string `json:"home_page_url"`
		HomePageNextURL     string `json:"home_page_next_url"`
		AtomURL             string `json:"atom_url"`
		HomePageLiteURL     string `json:"home_page_lite_url"`
		HomePageNextLiteURL string `json:"home_page_next_lite_url"`
		RemainingCount      int    `json:"remaining_count"`
		RemainingLabel      string `json:"remaining_label"`
		Items               []struct {
			Title             string `json:"title"`
			Summary           string `json:"summary"`
			ContentText       string `json:"content_text"`
			ContentHTML       string `json:"content_html"`
			ID                string `json:"id"`
			URL               string `json:"url"`
			DatePublished     string `json:"date_published"`
			DateModified      string `json:"date_modified"`
			OriginalPublished string `json:"_original_published"`
			OriginalLanguage  string `json:"_original_language"`
			Translations      struct {
				En struct {
					Title string `json:"title"`
				} `json:"en"`
				ZhHans struct {
					Title string `json:"title"`
				} `json:"zh-Hans"`
				Ja struct {
					Title string `json:"title"`
				} `json:"ja"`
				ZhHant struct {
					Title string `json:"title"`
				} `json:"zh-Hant"`
			} `json:"_translations"`
			Authors []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"authors"`
			Score       int `json:"_score"`
			NumComments int `json:"_num_comments"`
			Links       []struct {
				URL  string `json:"url"`
				Name string `json:"name"`
			} `json:"_links"`
			LiteContentHTML string `json:"_lite_content_html"`
			Author          struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"author"`
			SiteIdentifier string   `json:"_site_identifier"`
			HumanTime      string   `json:"_human_time"`
			Category       string   `json:"_category"`
			Order          int      `json:"order"`
			Image          string   `json:"image,omitempty"`
			Tags           []string `json:"tags,omitempty"`
			TitlePrefix    string   `json:"_title_prefix,omitempty"`
			TagLinks       []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"_tag_links,omitempty"`
		} `json:"items"`
	} `json:"_groups"`
}

func getBuzzingFeedEvery12Hours() BuzzingFeed {
	if !buzzingFeedLoadFromRemote {
		data, err := os.ReadFile("feed.json")
		if err != nil {
			log.Errorf("failed to read buzzing feed: %v", err)
			return BuzzingFeed{}
		}

		err = json.Unmarshal(data, &globalBuzzingFeed)
		if err != nil {
			log.Errorf("failed to unmarshal buzzing feed: %v", err)
			return BuzzingFeed{}
		}

		globalBuzzingFeedUpdatedAt = time.Now()
		return globalBuzzingFeed
	}

	if time.Since(globalBuzzingFeedUpdatedAt) < 4*time.Hour {
		return globalBuzzingFeed
	}

	// curl https://www.buzzing.cc/feed.json
	resp, err := http.Get("https://www.buzzing.cc/feed.json")
	if err != nil {
		log.Errorf("failed to get buzzing feed: %v", err)
		return BuzzingFeed{}
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read buzzing feed: %v", err)
		return BuzzingFeed{}
	}

	err = json.Unmarshal(body, &globalBuzzingFeed)
	if err != nil {
		log.Errorf("failed to unmarshal buzzing feed: %v", err)
		return BuzzingFeed{}
	}

	globalBuzzingFeedUpdatedAt = time.Now()
	return globalBuzzingFeed
}
