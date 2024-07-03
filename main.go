package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	humanize "github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
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
	}

	tmpl = template.Must(template.New("").Funcs(tmplFuncs).ParseFS(tmplFS, "tmpl/*.html"))

	CFTurnstileURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	port               = os.Getenv("PORT")
	CFTurnstileSecret  = os.Getenv("CF_SECRET")
	CFTurnstileSiteKey = os.Getenv("CF_SITEKEY")
	GHClientID         = os.Getenv("GH_CLIENT_ID")
	GHSecret           = os.Getenv("GH_SECRET")
	SiteURL            = os.Getenv("SITE_URL")
	DBPATH             = orenv("DBPATH", "rssy.db")
	TimeFormat         = "2006-01-02 15:04:05"

	GHRedirectURL = fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=user&redirect_uri=%s",
		GHClientID, fmt.Sprintf("%s/login/callback", SiteURL))

	CipherKey = []byte{}

	globalDB *gorm.DB

	fetchParseJob = FeedParseJob{
		jobs: sync.Map{},
		tk:   time.NewTicker(2 * time.Hour),
	}
)

func randCipherKey() {
	CipherKey = make([]byte, 32)
	_, err := rand.Read(CipherKey[:])
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	randCipherKey()
	initDB(DBPATH)

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
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Articles":           articles,
			"CFTurnstileSiteKey": CFTurnstileSiteKey,
		})
	})

	r.POST("/article/add", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		feedURL := c.PostForm("url")

		if email == "" || feedURL == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		err := addFeedAndCreateArticles(feedURL, email)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	r.GET("/article/:uid", checklogin, func(c *gin.Context) {
		uid := c.Param("uid")
		email := c.GetString("email")

		itemLink, err := getReadArticle(uid, email)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Redirect(http.StatusSeeOther, itemLink)
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

	log.Infof("Running on %s", SiteURL)
	r.Run(fmt.Sprintf(":%s", port))
}

func getReadArticle(uid, email string) (string, error) {
	article := Article{}

	err := globalDB.Where("uid = ? and email = ?", uid, email).First(&article).Error
	if err != nil {
		return "", fmt.Errorf("could not get article: %v", err)
	}

	err = globalDB.Model(article).Where("uid = ? and email = ?", uid, email).Update("read", true).Error
	if err != nil {
		return "", fmt.Errorf("could not read article: %v", err)
	}

	return article.Link, nil
}

func addFeedAndCreateArticles(feedURL, email string) error {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return fmt.Errorf("could not fetch feed: %v", err)
	}

	defer fetchParseJob.AddJob(feedURL, email)

	feedID, err := getSetFeed(feedURL, email)
	if err != nil {
		return fmt.Errorf("could not set feed: %v", err)
	}

	articles := make([]*Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		if !rssItemFilter(item, time.Hour*24*7) {
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
		return fmt.Errorf("could not create articles: %v", err)
	}

	return nil
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
	ID       int64  `json:"id" gorm:"column:id"`
	URL      string `json:"url" gorm:"column:url"`
	CreateAt int64  `json:"create_at" gorm:"column:create_at"`
	Priority int    `json:"priority" gorm:"column:priority"`
	Email    string `json:"email" gorm:"column:email"`
}

func getSetFeed(url, email string) (int64, error) {
	feed := &Feed{
		URL:      url,
		Email:    email,
		CreateAt: time.Now().Unix(),
		Priority: 1,
	}

	result := globalDB.Where("url = ? and email = ?", url, email).FirstOrCreate(feed)
	if result.Error != nil {
		return 0, result.Error
	}

	return feed.ID, nil
}

func initDB(filepath string) {
	db, err := gorm.Open(sqlite.Open(filepath), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal(err)
	}

	err = db.AutoMigrate(&Article{}, &Feed{})
	if err != nil {
		log.Fatal(err)
	}

	globalDB = db
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

func cfValidate(r *http.Request) bool {
	token := r.Form.Get("cf-turnstile-response")
	ip := r.Header.Get("CF-Connecting-IP")

	if token == "" || ip == "" {
		return false
	}

	form := url.Values{}
	form.Set("secret", CFTurnstileSecret)
	form.Set("response", token)
	form.Set("remoteip", ip)
	idempotencyKey := uuid.New().String()
	form.Set("idempotency_key", idempotencyKey)

	resp, err := http.PostForm(CFTurnstileURL, form)
	if err != nil {
		return false
	}

	type CFTurnstileResponse struct {
		Success bool `json:"success"`
	}

	var cfresp CFTurnstileResponse
	err = json.NewDecoder(resp.Body).Decode(&cfresp)

	return err != nil || cfresp.Success
}

func orenv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}

func rssItemFilter(item *gofeed.Item, dur time.Duration) bool {
	return item.PublishedParsed.Unix() > time.Now().Add(-dur).Unix()
}

type FeedParseJob struct {
	jobs sync.Map
	tk   *time.Ticker
}

type fetchJob struct {
	feedURL       string
	email         string
	lastFetchedAt time.Time
}

func (t *FeedParseJob) Start() {
	log.Infof("start feed parse job")
	for range t.tk.C {
		log.Infof("ticker to get feed, now: %v", time.Now())

		newmap := sync.Map{}

		t.jobs.Range(func(key, value interface{}) bool {
			job := value.(fetchJob)

			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(job.feedURL)
			if err != nil {
				log.Errorf("ticker to get feed error: %v", err)
			}

			feedFilter := func(item *gofeed.Item) bool {
				return item.PublishedParsed.After(job.lastFetchedAt)
			}

			articles := make([]*Article, 0, len(feed.Items))

			for _, item := range feed.Items {
				if !feedFilter(item) {
					continue
				}

				articles = append(articles, &Article{
					Uid:       uuid.New().String(),
					Name:      feed.Title,
					FeedID:    0,
					Email:     job.email,
					Title:     item.Title,
					Link:      item.Link,
					Read:      false,
					Hide:      false,
					Deleted:   false,
					Content:   item.Content,
					PublishAt: item.PublishedParsed.Unix(),
				})
			}

			if err := globalDB.CreateInBatches(articles, 10).Error; err != nil {
				log.Errorf("could not create articles: %v", err)
			}

			log.Infof("url:%s, email:%s, ticker fetched %d articles", job.feedURL, job.email, len(articles))

			newmap.Store(key, fetchJob{
				feedURL:       job.feedURL,
				email:         job.email,
				lastFetchedAt: time.Now(),
			})

			return true
		})

		t.jobs = newmap
	}
}

func (t *FeedParseJob) Stop() {
	t.tk.Stop()
}

func (t *FeedParseJob) AddJob(feedURL, email string) {
	t.jobs.Store(email+"-"+feedURL, fetchJob{
		feedURL:       feedURL,
		email:         email,
		lastFetchedAt: time.Now(),
	})
}
