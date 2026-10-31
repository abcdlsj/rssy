package internal

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func ServerRouter() *gin.Engine {
	r := gin.Default()

	r.SetFuncMap(tmplFuncs)
	r.SetHTMLTemplate(tmpl)

	// 从本地缓存恢复 RSSHub 注册表，并在后台按需刷新
	loadExploreCache()
	ensureExploreData()

	checklogin := func(c *gin.Context) {
		// 调试模式下直接使用默认邮箱 or 检查是否启用了 GitHub 登录
		if DebugMode || !checkAnyUserHasGitHubLogin() {
			c.Set("email", DefaultEmail)
			c.Next()
			return
		}

		// 启用了 GitHub 登录，检查是否有有效的 GitHub 会话
		session, err := checkRefreshGHStatus(c.Request)
		if err != nil {
			// 没有有效会话，重定向到登录页面
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}

		c.Set("email", session.Email)
		c.Next()
	}

	// 页面里用 fetch 调用时带上这个头，接口返回 JSON 而不是重定向
	wantsJSON := func(c *gin.Context) bool {
		return c.GetHeader("X-Requested-With") == "fetch"
	}

	serveAsset := func(name, contentType string) gin.HandlerFunc {
		return func(c *gin.Context) {
			data, err := assetFs.ReadFile("assets/" + name)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Header("Cache-Control", "public, max-age=86400")
			c.Data(http.StatusOK, contentType, data)
		}
	}
	r.GET("/app.css", serveAsset("app.css", "text/css; charset=utf-8"))
	r.GET("/app.js", serveAsset("app.js", "application/javascript; charset=utf-8"))

	r.GET("/", checklogin, func(c *gin.Context) {
		email := c.GetString("email")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		articles := getRecentlyArticles(email)
		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles":            articles,
			"SiteURL":             SiteURL,
			"Headline":            "Unreads",
			"ShowHidden":          c.Query("show_hidden") == "true",
			"DisplayHiddenToggle": true,
			"IsHome":              true,
			"Categories":          getCategories(email),
		})
	})

	r.GET("/feed", checklogin, func(c *gin.Context) {
		email := c.GetString("email")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		feeds := getFeeds(email)
		categories := getCategories(email)
		currentCategory := c.Query("category")

		categoryFeeds := make(map[string][]Feed)
		for _, cat := range categories {
			categoryFeeds[cat.Name] = getFeedsByCategory(email, cat.Name)
		}

		c.HTML(http.StatusOK, "feed.html", gin.H{
			"Feeds":           feeds,
			"Categories":      categories,
			"AllFeeds":        feeds,
			"CategoryFeeds":   categoryFeeds,
			"CurrentCategory": currentCategory,
			"SiteURL":         SiteURL,
			"InboxFeeds":      getFeedsByCategory(email, ""),
			"UnreadCounts":    getUnreadCounts(email),
		})
	})

	r.GET("/category/:name", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		category := c.Param("name")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		articles := getArticlesByCategory(email, category)
		headline := category
		if category == "" {
			headline = "Inbox"
		}

		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles":   articles,
			"SiteURL":    SiteURL,
			"Headline":   headline,
			"IsCategory": true,
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
		categories := getCategories(email)
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
			"Categories":    categories,
			"Feed":          feed,
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
		if wantsJSON(c) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
		c.Redirect(http.StatusFound, "/feed/"+id)
	})

	r.POST("/feed/:id/update", checklogin, func(c *gin.Context) {
		hide := c.PostForm("hide_unread") == "true"
		enableReadability := c.PostForm("enable_readability") == "true"
		highlight := c.PostForm("highlight") == "true"
		category := c.PostForm("category")

		email := c.GetString("email")
		id := c.Param("id")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		log.Infof("update feed: %s, %t, %t, %t, %s", id, hide, enableReadability, highlight, category)

		if err := updateFeed(email, id, hide, enableReadability, highlight); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if c.PostForm("set_category") == "true" || category != "" {
			updateFeedCategory(email, id, category)
		}
		if wantsJSON(c) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
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
		from := c.Query("from")
		if from != "favorites" {
			from = "home"
		}
		c.HTML(http.StatusOK, "content.html", gin.H{
			"SiteURL":   SiteURL,
			"Article":   article,
			"Title":     article.Title,
			"PublishAt": article.PublishAt,
			"Content":   article.Content,
			"From":      from,
			"Nav":       getReaderNav(article, email, from),
		})
	})

	// 标记已读 / 未读（列表页的 done 和撤销）
	r.POST("/article/:uid/mark", checklogin, func(c *gin.Context) {
		uid := c.Param("uid")
		email := c.GetString("email")
		if email == "" || uid == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}
		read := c.PostForm("read") != "false"
		if err := setArticleRead(uid, email, read); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if wantsJSON(c) {
			c.JSON(http.StatusOK, gin.H{"read": read})
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
	})

	r.POST("/articles/mark", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}
		read := c.PostForm("read") != "false"
		n, err := setArticlesRead(c.PostFormArray("uid"), email, read)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if wantsJSON(c) {
			c.JSON(http.StatusOK, gin.H{"count": n, "read": read})
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
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

	r.POST("/article/:uid/favorite", checklogin, func(c *gin.Context) {
		uid := c.Param("uid")
		email := c.GetString("email")

		if email == "" || uid == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		err := toggleFavorite(uid, email)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if wantsJSON(c) {
			c.JSON(http.StatusOK, gin.H{"favorite": isArticleFavorite(uid, email)})
			return
		}

		c.Redirect(http.StatusSeeOther, c.Request.Referer())
	})

	r.GET("/favorites", checklogin, func(c *gin.Context) {
		email := c.GetString("email")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		articles := getFavoriteArticles(email)
		c.HTML(http.StatusOK, "articles.html", gin.H{
			"Articles": articles,
			"SiteURL":  SiteURL,
			"Headline": "Favorites",
		})
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		favicon, _ := assetFs.ReadFile("assets/favicon.ico")
		c.Data(http.StatusOK, "image/x-icon", favicon)
	})

	serveIcon := func(name, contentType string) gin.HandlerFunc {
		return func(c *gin.Context) {
			data, err := assetFs.ReadFile("assets/icons/" + name)
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Header("Cache-Control", "public, max-age=604800")
			c.Data(http.StatusOK, contentType, data)
		}
	}
	// PWA 图标与 Safari 添加到主屏时使用的图标
	r.GET("/icon-192.png", serveIcon("icon-192.png", "image/png"))
	r.GET("/icon-512.png", serveIcon("icon-512.png", "image/png"))
	r.GET("/icon-maskable-512.png", serveIcon("icon-maskable-512.png", "image/png"))
	r.GET("/apple-touch-icon.png", serveIcon("apple-touch-icon.png", "image/png"))
	r.GET("/favicon-32x32.png", serveIcon("favicon-32.png", "image/png"))

	r.GET("/manifest.webmanifest", func(c *gin.Context) {
		c.Header("Content-Type", "application/manifest+json")
		c.Header("Cache-Control", "public, max-age=3600")
		c.JSON(http.StatusOK, gin.H{
			"name":             "RSSy",
			"short_name":       "RSSy",
			"description":      "Personal RSS reader",
			"start_url":        "/",
			"scope":            "/",
			"display":          "standalone",
			"orientation":      "portrait",
			"background_color": "#ffffff",
			"theme_color":      "#ffffff",
			"icons": []gin.H{
				{"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"},
				{"src": "/icon-512.png", "sizes": "512x512", "type": "image/png"},
				{"src": "/icon-maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
			},
		})
	})

	r.GET("/login", func(c *gin.Context) {
		// 使用管理员用户的 GitHub 登录配置
		adminPref, err := getAdminPreference()
		if err != nil || !adminPref.EnableGitHubLogin {
			c.String(http.StatusInternalServerError, "<html><body><h1>GitHub login not enabled</h1></body></html>")
			return
		}

		redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=user&redirect_uri=%s",
			adminPref.GitHubClientID, fmt.Sprintf("%s/login/callback", SiteURL))
		c.Redirect(http.StatusSeeOther, redirectURL)
	})

	r.GET("/login/callback", func(c *gin.Context) {
		code := c.Query("code")

		// 使用管理员用户的 GitHub 登录配置
		adminPref, err := getAdminPreference()
		if err != nil || !adminPref.EnableGitHubLogin {
			c.String(http.StatusInternalServerError, "<html><body><h1>GitHub login not enabled</h1></body></html>")
			return
		}

		ak, sk, expiresIn := getGithubAccessToken(code, "", adminPref.GitHubClientID, adminPref.GitHubSecret)
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

	r.GET("/explore", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		ensureExploreData()
		data, loading, lastErr := getExploreSnapshot()

		query := strings.TrimSpace(c.Query("q"))
		category := strings.TrimSpace(c.Query("category"))
		if category == "" {
			category = "featured"
		}
		page, _ := strconv.Atoi(c.Query("page"))

		pageData := ExplorePage{
			SiteURL:    SiteURL,
			Category:   category,
			Query:      query,
			Page:       page,
			Loading:    loading,
			RSSHubURL:  exploreBaseURL(),
			PageSize:   explorePageSize,
			Categories: []ExploreCategory{},
			Items:      []ExploreItem{},
		}
		pageData.Back = exploreURL(query, category, page)

		if c.Query("added") != "" {
			pageData.Notice = "Subscribed: " + c.Query("added")
		}
		if message := c.Query("error"); message != "" {
			pageData.ErrorMessage = message
		}

		if data == nil {
			if lastErr != "" {
				pageData.ErrorMessage = "Could not load RSSHub routes: " + lastErr
			}
			c.HTML(http.StatusOK, "explore.html", pageData)
			return
		}

		subscribed := getSubscribedFeedURLs(email)
		items := filterExploreItems(data.Items, query, category)
		for i := range items {
			if _, ok := subscribed[normalizeFeedURL(items[i].Link)]; ok {
				items[i].Added = true
			}
		}

		paged, currentPage, totalPages := paginateExplore(items, page)
		pageData.Items = paged
		pageData.Total = len(items)
		pageData.Page = currentPage
		pageData.TotalPages = totalPages
		pageData.HasPrev = currentPage > 1
		pageData.HasNext = currentPage < totalPages
		pageData.PrevURL = exploreURL(query, category, currentPage-1)
		pageData.NextURL = exploreURL(query, category, currentPage+1)
		pageData.RouteCount = data.RouteCount
		pageData.UpdatedAt = data.UpdatedAt
		pageData.Categories = data.Categories

		c.HTML(http.StatusOK, "explore.html", pageData)
	})

	r.POST("/explore/add", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		feedURL := strings.TrimSpace(c.PostForm("url"))
		back := c.PostForm("back")

		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}
		if !strings.HasPrefix(back, "/explore") {
			back = "/explore"
		}

		if feedURL == "" || !strings.HasPrefix(feedURL, exploreBaseURL()) {
			c.Redirect(http.StatusSeeOther, back+exploreJoin(back)+"error="+url.QueryEscape("Not a valid RSSHub URL"))
			return
		}

		if _, err := addFeedAndCreateArticles(feedURL, email); err != nil {
			log.Errorf("add explore feed %s: %v", feedURL, err)
			message := "Could not subscribe. RSSHub may not be able to fetch this route right now, try again later: " + err.Error()
			c.Redirect(http.StatusSeeOther, back+exploreJoin(back)+"error="+url.QueryEscape(message))
			return
		}

		name := strings.TrimSpace(c.PostForm("name"))
		c.Redirect(http.StatusSeeOther, back+exploreJoin(back)+"added="+url.QueryEscape(name))
	})

	// 订阅前在 RSSy 里预览 RSSHub 路由的条目
	r.GET("/explore/preview", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		feedURL := strings.TrimSpace(c.Query("url"))
		back := c.Query("back")
		if !strings.HasPrefix(back, "/explore") || strings.HasPrefix(back, "/explore/preview") {
			back = "/explore"
		}

		values := url.Values{}
		values.Set("url", feedURL)
		if name := c.Query("name"); name != "" {
			values.Set("name", name)
		}
		if ns := c.Query("ns"); ns != "" {
			values.Set("ns", ns)
		}
		values.Set("back", back)

		page := ExplorePreviewPage{
			SiteURL:   SiteURL,
			FeedURL:   feedURL,
			Name:      c.Query("name"),
			Namespace: c.Query("ns"),
			Back:      back,
			Self:      "/explore/preview?" + values.Encode(),
		}
		if added := c.Query("added"); added != "" {
			page.Notice = "Subscribed: " + added
		}
		if message := c.Query("error"); message != "" {
			page.ErrorMessage = message
		}

		if !isExploreFeedURL(feedURL) {
			page.ErrorMessage = "Only routes from " + exploreBaseURL() + " can be previewed."
			c.HTML(http.StatusBadRequest, "explore-preview.html", page)
			return
		}

		if id, ok := getSubscribedFeedURLs(email)[normalizeFeedURL(feedURL)]; ok {
			page.Added = true
			page.FeedID = id
		}

		preview, err := fetchExplorePreview(feedURL, c.Query("refresh") == "true")
		if err != nil {
			log.Errorf("preview explore feed %s: %v", feedURL, err)
			if page.ErrorMessage == "" {
				page.ErrorMessage = "RSSHub could not build this route right now. It may need extra config or be rate-limited. " + err.Error()
			}
		}
		page.Preview = preview
		c.HTML(http.StatusOK, "explore-preview.html", page)
	})

	r.POST("/explore/refresh", checklogin, func(c *gin.Context) {
		forceRefreshExplore()
		c.Redirect(http.StatusSeeOther, "/explore")
	})

	r.GET("/preference", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		pref, err := getUserPreference(email)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get preferences: %v", err)
			return
		}

		categories := getCategories(email)
		message := c.Query("message")

		c.HTML(http.StatusOK, "preference.html", gin.H{
			"SiteURL":    SiteURL,
			"Preference": pref,
			"IsAdmin":    isAdminUser(email),
			"Categories": categories,
			"Message":    message,
		})
	})

	r.POST("/category/add", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		name := c.PostForm("name")
		color := c.PostForm("color")

		if email == "" || name == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		if color == "" {
			color = "#007bff"
		}

		err := createCategory(name, color, email)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create category: %v", err)
			return
		}

		c.Redirect(http.StatusFound, "/preference")
	})

	r.POST("/category/delete/:name", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		name := c.Param("name")

		if email == "" || name == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		err := deleteCategory(email, name)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to delete category: %v", err)
			return
		}

		c.Redirect(http.StatusFound, "/preference")
	})

	r.POST("/feed/:id/category", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		id := c.Param("id")
		category := c.PostForm("category")

		if email == "" || id == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		err := updateFeedCategory(email, id, category)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to update feed category: %v", err)
			return
		}
		if wantsJSON(c) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}

		currentCategory := c.PostForm("current_category")
		if currentCategory == "" {
			c.Redirect(http.StatusFound, "/feed")
		} else {
			c.Redirect(http.StatusFound, "/feed?category="+currentCategory)
		}
	})

	r.POST("/preference/update", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		action := c.PostForm("action")
		var message string

		switch action {
		case "cleanup_expired":
			days, err := strconv.Atoi(c.PostForm("cleanup_expired_days"))
			if err != nil || days <= 0 {
				days = 30
			}
			deleted, err := cleanupExpiredArticles(email, days)
			if err != nil {
				message = fmt.Sprintf("Cleanup failed: %v", err)
			} else {
				message = fmt.Sprintf("Deleted %d expired articles", deleted)
			}

		case "cleanup_read":
			deleted, err := cleanupReadArticles(email)
			if err != nil {
				message = fmt.Sprintf("Cleanup failed: %v", err)
			} else {
				message = fmt.Sprintf("Deleted %d read articles", deleted)
			}

		case "save":
			pref, err := getUserPreference(email)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to get preferences: %v", err)
				return
			}

			pref.CleanupExpiredDays, _ = strconv.Atoi(c.PostForm("cleanup_expired_days"))
			if pref.CleanupExpiredDays <= 0 {
				pref.CleanupExpiredDays = 30
			}

			pref.EnableAutoCleanup = c.PostForm("enable_auto_cleanup") == "on"
			pref.EnableNotification = c.PostForm("enable_notification") == "on"
			pref.NotificationTime = c.PostForm("notification_time")
			pref.SendCloudAPIUser = c.PostForm("sendcloud_api_user")
			pref.SendCloudAPIKey = c.PostForm("sendcloud_api_key")
			pref.SendCloudFrom = c.PostForm("sendcloud_from")
			pref.SendCloudFromName = c.PostForm("sendcloud_from_name")
			pref.EnableAISummary = c.PostForm("enable_ai_summary") == "on"
			pref.AISummaryTime = c.PostForm("ai_summary_time")
			pref.AISummaryPrompt = c.PostForm("ai_summary_prompt")

			// Admin-only settings
			if isAdminUser(email) {
				log.Infof("Processing admin settings for user: %s", email)
				log.Infof("enable_github_login form value: %s", c.PostForm("enable_github_login"))
				pref.EnableGitHubLogin = c.PostForm("enable_github_login") == "on"
				pref.GitHubClientID = c.PostForm("github_client_id")
				pref.GitHubSecret = c.PostForm("github_secret")
				pref.OpenAIAPIKey = c.PostForm("openai_api_key")
				pref.OpenAIEndpoint = c.PostForm("openai_endpoint")
				log.Infof("Set EnableGitHubLogin to: %t", pref.EnableGitHubLogin)
			} else {
				log.Infof("User %s is not admin, skipping admin settings", email)
			}

			err = updateUserPreference(email, pref)
			if err != nil {
				message = fmt.Sprintf("Failed to save preferences: %v", err)
			} else {
				message = "Preferences saved successfully"
			}
		}

		c.Redirect(http.StatusFound, "/preference?message="+message)
	})

	r.GET("/ai-summary", checklogin, func(c *gin.Context) {
		email := c.GetString("email")
		if email == "" {
			c.String(http.StatusBadRequest, "invalid request")
			return
		}

		summaries, err := getAISummariesForUser(email, 30)
		if err != nil {
			log.Errorf("Failed to get AI summaries: %v", err)
			summaries = []AISummary{}
		}

		pref, _ := getUserPreference(email)
		c.HTML(http.StatusOK, "ai-summary.html", gin.H{
			"SiteURL":    SiteURL,
			"Summaries":  summaries,
			"Preference": pref,
			"Today":      time.Now().In(TimeZone).Format("2006-01-02"),
		})
	})

	return r
}
