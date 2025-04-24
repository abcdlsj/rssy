package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

func sendmsg() {
	url := fmt.Sprintf("https://www.notifyx.cn/api/v1/send/%s", NotiKey)

	articles, err := getYesterdayHighlightedUnreadArticles()
	if len(articles) == 0 || err != nil {
		log.Errorf("getYesterdayHighlightedUnreadArticles failed, err: %s", err)
		return
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString(fmt.Sprintf("昨日（%s）未读且高亮的 RSS 文章：\n\n",
		time.Now().Add(-24*time.Hour).Format("2006-01-02")))

	for _, article := range articles {
		contentBuilder.WriteString(fmt.Sprintf("- [%s](%s)\n", article.Title, article.Link))
	}

	message := map[string]string{
		"title":       "每日 RSS 摘要",
		"content":     contentBuilder.String(),
		"description": fmt.Sprintf("共 %d 篇文章", len(articles)),
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	log.Infof("notifyx result %v", result)
}
