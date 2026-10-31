package internal

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

// Explore 预览：订阅前先在 RSSy 里看看这个 RSSHub 路由会产出哪些条目。

const (
	explorePreviewTTL     = 10 * time.Minute
	explorePreviewTimeout = 25 * time.Second
	explorePreviewLimit   = 50
	explorePreviewSnippet = 220
)

type ExplorePreviewItem struct {
	Title     string
	Link      string
	Author    string
	Published int64
	Snippet   string
}

type ExplorePreview struct {
	Title       string
	Description string
	SiteLink    string
	Items       []ExplorePreviewItem
	FetchedAt   int64
}

type explorePreviewEntry struct {
	preview *ExplorePreview
	at      time.Time
}

var (
	explorePreviewMu    sync.Mutex
	explorePreviewCache = map[string]explorePreviewEntry{}

	htmlTagRe = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>|<[^>]+>`)
)

// isExploreFeedURL 只允许预览本实例 RSSHub 下的地址，避免变成任意 URL 代理
func isExploreFeedURL(feedURL string) bool {
	base := exploreBaseURL()
	return base != "" && strings.HasPrefix(feedURL, base+"/")
}

func fetchExplorePreview(feedURL string, force bool) (*ExplorePreview, error) {
	explorePreviewMu.Lock()
	if entry, ok := explorePreviewCache[feedURL]; ok && !force && time.Since(entry.at) < explorePreviewTTL {
		explorePreviewMu.Unlock()
		return entry.preview, nil
	}
	explorePreviewMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), explorePreviewTimeout)
	defer cancel()

	feed, err := gofeed.NewParser().ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load this route: %v", err)
	}

	preview := &ExplorePreview{
		Title:       strings.TrimSpace(feed.Title),
		Description: plainText(feed.Description, 300),
		SiteLink:    feed.Link,
		FetchedAt:   time.Now().Unix(),
	}

	for i, item := range feed.Items {
		if i >= explorePreviewLimit {
			break
		}
		if item == nil {
			continue
		}
		p := ExplorePreviewItem{
			Title: strings.TrimSpace(item.Title),
			Link:  item.Link,
		}
		if p.Title == "" {
			p.Title = "(untitled)"
		}
		if item.Author != nil {
			p.Author = item.Author.Name
		} else if len(item.Authors) > 0 && item.Authors[0] != nil {
			p.Author = item.Authors[0].Name
		}
		if item.PublishedParsed != nil {
			p.Published = item.PublishedParsed.Unix()
		} else if item.UpdatedParsed != nil {
			p.Published = item.UpdatedParsed.Unix()
		}
		body := item.Description
		if strings.TrimSpace(body) == "" {
			body = item.Content
		}
		p.Snippet = plainText(body, explorePreviewSnippet)
		if p.Snippet == p.Title {
			p.Snippet = ""
		}
		preview.Items = append(preview.Items, p)
	}

	explorePreviewMu.Lock()
	// 顺手清掉过期条目，防止缓存无限增长
	for k, v := range explorePreviewCache {
		if time.Since(v.at) > explorePreviewTTL {
			delete(explorePreviewCache, k)
		}
	}
	explorePreviewCache[feedURL] = explorePreviewEntry{preview: preview, at: time.Now()}
	explorePreviewMu.Unlock()

	return preview, nil
}

// plainText 把条目里的 HTML 转成一段纯文本摘要
func plainText(s string, limit int) string {
	text := htmlTagRe.ReplaceAllString(s, " ")
	text = html.UnescapeString(text)
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

// ExplorePreviewPage 是预览模板的渲染数据
type ExplorePreviewPage struct {
	SiteURL      string
	FeedURL      string
	Name         string
	Namespace    string
	Back         string
	Self         string
	Added        bool
	FeedID       int64
	Preview      *ExplorePreview
	ErrorMessage string
	Notice       string
}
