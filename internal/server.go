package internal

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func ServerRouter() *gin.Engine {
	r := gin.Default()

	r.SetFuncMap(tmplFuncs)
	r.SetHTMLTemplate(tmpl)

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
				"hide_unread":        strconv.FormatBool(feed.HideUnread),
				"enable_readability": strconv.FormatBool(feed.EnableReadability),
				"highlight":          strconv.FormatBool(feed.Highlight),
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
		enableReadability := c.PostForm("enable_readability") == "true"
		highlight := c.PostForm("highlight") == "true"

		email := c.GetString("email")
		id := c.Param("id")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		log.Infof("update feed: %s, %t, %t, %t", id, hide, enableReadability, highlight)

		updateFeed(email, id, hide, enableReadability, highlight)
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

	r.GET("/article/:uid/delete", checklogin, func(c *gin.Context) {
		uid := c.Param("uid")
		email := c.GetString("email")

		if email == "" || uid == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		err := deleteArticle(uid, email)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

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

	return r
}
