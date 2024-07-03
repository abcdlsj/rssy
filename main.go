package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	//go:embed tmpl/*.html
	tmplFS embed.FS

	//go:embed assets/*
	assetFs embed.FS

	dbFile = "rssy.db"

	tmplFuncs = template.FuncMap{
		"truncate": func(content string, length int) string {
			if len(content) <= length {
				return content
			}
			return content[:length]
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
	TimeFormat         = "2006-01-02 15:04:05"

	GHRedirectURL = fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=user&redirect_uri=%s",
		GHClientID, fmt.Sprintf("%s/login/callback", SiteURL))

	CipherKey = []byte{}

	globalDB *gorm.DB
)

func randCipherKey() {
	CipherKey = make([]byte, 32)
	_, err := rand.Read(CipherKey[:])
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	randCipherKey()

	flag.StringVar(&dbFile, "db", dbFile, "database file")
	flag.Parse()

	initDB(dbFile)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.SetFuncMap(tmplFuncs)
	r.SetHTMLTemplate(tmpl)

	checklogin := func(c *gin.Context) {
		session, err := checkRefreshGHStatus(c.Writer, c.Request)
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		c.Set("session", session)
	}

	r.GET("/", checklogin, func(c *gin.Context) {
		articles := getRecentlyArticles()
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Articles":           articles,
			"CFTurnstileSiteKey": CFTurnstileSiteKey,
		})
	})

	r.POST("/article/add", checklogin, func(c *gin.Context) {
		session := c.MustGet("session").(*Session)

		feedURL := c.PostForm("url")
		articles, err := AddFeed(feedURL, session.Email)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch feed")
			return
		}

		globalDB.CreateInBatches(articles, 10)

		c.Redirect(http.StatusSeeOther, "/")
	})

	r.POST("/article/:uid", func(c *gin.Context) {
		uid := c.Param("uid")
		readArticle(uid)
		c.Redirect(http.StatusSeeOther, "/")
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
		if login != "" {
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

	log.Printf("Running on %s", SiteURL)
	r.Run(fmt.Sprintf(":%s", port))
}

func AddFeed(feedURL, email string) ([]*Article, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("could not fetch feed: %v", err)
	}

	feedID, err := getSetFeed(feedURL, email)
	if err != nil {
		return nil, fmt.Errorf("could not set feed: %v", err)
	}

	var articles []*Article
	for _, item := range feed.Items {
		articles = append(articles, &Article{
			Uuid:      uuid.New().String(),
			Name:      feed.Title,
			FeedID:    feedID,
			Title:     item.Title,
			Link:      item.Link,
			Read:      false,
			Hide:      false,
			Deleted:   false,
			Content:   item.Content,
			PublishAt: item.PublishedParsed.Format(TimeFormat),
			CreateAt:  time.Now().Unix(),
		})
	}

	return articles, nil
}

type Session struct {
	AK     string `json:"ak"`
	RK     string `json:"rk"`
	Expire int    `json:"ak_expire"`
	Email  string `json:"email"`
}

func checkRefreshGHStatus(w http.ResponseWriter, r *http.Request) (*Session, error) {
	session := getCookieSession(r)
	if session == nil {
		return nil, fmt.Errorf("browser session is nil")
	}

	log.Printf("get session: %+v", session)
	if time.Now().Unix() > int64(session.Expire) {
		if session.RK == "" {
			return nil, fmt.Errorf("refresh token is empty")
		}
		ak, sk, expiresIn := getGithubAccessToken("", session.RK)
		if ak == "" {
			return nil, fmt.Errorf("failed to get access token")
		}

		login, email := getGithubData(ak)
		if login != "" {
			return nil, fmt.Errorf("failed to get github data")
		}

		session = &Session{
			AK:     ak,
			RK:     sk,
			Expire: int(time.Now().Unix()) + expiresIn,
			Email:  email,
		}

		setCookieSession(w, "s", *session)
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
		log.Printf("decrypt session error: %v", err)
		return nil
	}

	return &session
}

func setCookieSession(w http.ResponseWriter, name string, session Session) {
	encryptSess, err := encryptSession(session)
	if err != nil {
		log.Printf("encrypt session error: %v", err)
		return
	}

	cookie := http.Cookie{
		Name:   name,
		Value:  encryptSess,
		MaxAge: 24 * 60 * 60 * 7,
		Path:   "/",
	}

	log.Printf("set cookie: %s, session: %+v\n", cookie.String(), session)
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
		log.Printf("Error: %s\n", err)
		return "", "", 0
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Printf("Error: %s\n", resperr)
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
		log.Printf("Error: %s\n", err)
		return "", "", 0
	}

	log.Printf("Github: %+v", ghresp)
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

	log.Printf("github data: %+v", ghresp)
	return ghresp.Login, ghresp.Email
}

type Article struct {
	Uuid string `json:"uid" gorm:"column:id"`
	Name string `json:"name" gorm:"column:name"`

	FeedID int64 `json:"feed_id" gorm:"column:feed_id"`

	Title string `json:"title" gorm:"column:title"`
	Link  string `json:"link" gorm:"column:link"`

	Read    bool `json:"read" gorm:"column:read"`
	Hide    bool `json:"hide" gorm:"column:hide"`
	Deleted bool `json:"deleted" gorm:"column:deleted"`

	CreateAt  int64  `json:"create_at" gorm:"column:create_at"`
	PublishAt string `json:"publish_at" gorm:"column:publish_at"`

	Content string `json:"content" gorm:"column:content"`
}

type Feed struct {
	ID       int64  `json:"id" gorm:"column:id,primarykey"`
	Site     string `json:"site" gorm:"column:site"`
	URL      string `json:"url" gorm:"column:url"`
	CreateAt int64  `json:"create_at" gorm:"column:create_at"`
	Priority int    `json:"priority" gorm:"column:priority"`
	User     string `json:"user" gorm:"column:user"`
}

func getSetFeed(url, email string) (int64, error) {
	feed := &Feed{}

	result := globalDB.Where("url = ? and user = ?", url, email).FirstOrCreate(feed)
	if result.Error != nil {
		return 0, result.Error
	}

	return feed.ID, nil
}

func initDB(filepath string) {
	db, err := gorm.Open(sqlite.Open(filepath), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	err = globalDB.AutoMigrate(&Article{}, &Feed{})
	if err != nil {
		log.Fatal(err)
	}

	globalDB = db
}

func readArticle(uid string) error {
	return globalDB.Model(&Article{}).Where("uid = ?", uid).Update("read", true).Error
}

func getRecentlyArticles() []Article {
	var articles []Article
	result := globalDB.Order("publish_at desc").Find(&articles)
	if result.Error != nil {
		log.Println(result.Error)
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
