package internal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

func scheduleSendDailyNotify(email string) {
	pref, err := getUserPreference(email)
	if err != nil {
		log.Errorf("Failed to get user preference for %s: %v", email, err)
		return
	}

	if pref.SendCloudAPIUser == "" || pref.SendCloudAPIKey == "" || pref.SendCloudFrom == "" {
		log.Errorf("SendCloud not configured for user %s", email)
		return
	}

	if pref.SendCloudFromName == "" {
		pref.SendCloudFromName = "RSSy"
	}

	articles, err := getYesterdayHighlightedUnreadArticlesForUser(email)
	if len(articles) == 0 || err != nil {
		log.Errorf("getYesterdayHighlightedUnreadArticlesForUser failed for %s, err: %s", email, err)
		return
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString(fmt.Sprintf("<h2>昨日（%s）未读且高亮的 RSS 文章</h2>",
		time.Now().Add(-24*time.Hour).Format("2006-01-02")))
	contentBuilder.WriteString(fmt.Sprintf("<p>共 %d 篇文章</p>", len(articles)))
	contentBuilder.WriteString("<ul>")

	for _, article := range articles {
		contentBuilder.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", article.Link, article.Title))
	}

	contentBuilder.WriteString("</ul>")

	postParams := url.Values{}
	postParams.Set("apiUser", pref.SendCloudAPIUser)
	postParams.Set("apiKey", pref.SendCloudAPIKey)
	postParams.Set("from", pref.SendCloudFrom)
	postParams.Set("fromName", pref.SendCloudFromName)
	postParams.Set("to", email)
	postParams.Set("subject", fmt.Sprintf("每日 RSS 摘要 - %s", time.Now().Add(-24*time.Hour).Format("2006-01-02")))
	postParams.Set("html", contentBuilder.String())

	requestURI := "http://api.sendcloud.net/apiv2/mail/send"
	postBody := bytes.NewBufferString(postParams.Encode())
	responseHandler, err := http.Post(requestURI, "application/x-www-form-urlencoded", postBody)
	if err != nil {
		log.Errorf("Failed to send email: %v", err)
		return
	}
	defer responseHandler.Body.Close()

	bodyBytes, err := ioutil.ReadAll(responseHandler.Body)
	if err != nil {
		log.Errorf("Failed to read response: %v", err)
		return
	}

	log.Infof("SendCloud result for %s: %s", email, string(bodyBytes))
}
