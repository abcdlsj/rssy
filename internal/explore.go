package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
)

// RSSHub Explore：拉取 RSSHub 实例的命名空间/路由注册表，缓存后供页面浏览与一键订阅。

const (
	exploreCacheScene = "explore"
	exploreCacheKey   = "namespaces"
	exploreTTL        = 12 * time.Hour
	explorePageSize   = 50
	exploreDescLimit  = 180
)

// 首页「推荐」展示的命名空间
var exploreFeaturedNamespaces = []string{
	"github", "bilibili", "zhihu", "weibo", "sspai", "v2ex", "xiaoyuzhou",
	"douban", "youtube", "telegram", "twitter", "xiaoheihe", "gcores",
	"nasa", "steam", "indienova", "jike", "hackernews", "producthunt",
	"ithome", "36kr", "readhub", "wallstreetcn", "pixiv",
}

var exploreCategoryLabels = map[string]string{
	"programming":       "Programming",
	"social-media":      "Social media",
	"new-media":         "New media",
	"traditional-media": "Traditional media",
	"blog":              "Blogs",
	"picture":           "Pictures",
	"reading":           "Reading",
	"game":              "Games",
	"anime":             "Anime",
	"university":        "Universities",
	"finance":           "Finance",
	"government":        "Government",
	"multimedia":        "Media",
	"bbs":               "Forums",
	"program-update":    "Software updates",
	"shopping":          "Shopping",
	"forecast":          "Forecasts",
	"travel":            "Travel",
	"design":            "Design",
	"live":              "Live",
	"sport":             "Sports",
	"study":             "Study",
	"journal":           "Journals",
	"other":             "Other",
}

// RSSHub /api/namespace 的返回结构
type exploreRawRoute struct {
	Path        json.RawMessage `json:"path"`
	Name        string          `json:"name"`
	Example     string          `json:"example"`
	Description string          `json:"description"`
	Categories  []string        `json:"categories"`
	View        int             `json:"view"`
	// requireConfig 既可能是 bool，也可能是配置项名称数组
	Features struct {
		RequireConfig    json.RawMessage `json:"requireConfig"`
		RequirePuppeteer json.RawMessage `json:"requirePuppeteer"`
		AntiCrawler      json.RawMessage `json:"antiCrawler"`
	} `json:"features"`
}

type exploreRawNamespace struct {
	Name   string                     `json:"name"`
	URL    string                     `json:"url"`
	Lang   string                     `json:"lang"`
	Routes map[string]exploreRawRoute `json:"routes"`
}

type ExploreItem struct {
	Key           string
	Namespace     string
	NamespaceName string
	Name          string
	Example       string
	Link          string
	Description   string
	Categories    []string
	RequiresSetup bool
	AntiCrawler   bool
	View          int
	Added         bool
}

type ExploreCategory struct {
	Name  string
	Label string
	Count int
}

type ExploreData struct {
	Items      []ExploreItem
	Categories []ExploreCategory
	RouteCount int
	UpdatedAt  int64
}

// ExploreCache 把注册表落库，避免每次重启都要重新拉取 3MB 数据
type ExploreCache struct {
	ID        int64  `json:"id" gorm:"primaryKey;column:id"`
	Key       string `json:"key" gorm:"column:key;uniqueIndex"`
	Data      string `json:"data" gorm:"column:data;type:text"`
	UpdatedAt int64  `json:"updated_at" gorm:"column:updated_at"`
}

var (
	exploreMu       sync.RWMutex
	exploreData     *ExploreData
	exploreLoading  atomic.Bool
	exploreLastErr  string
	exploreFeatured map[string]bool
)

func init() {
	exploreFeatured = make(map[string]bool, len(exploreFeaturedNamespaces))
	for _, ns := range exploreFeaturedNamespaces {
		exploreFeatured[ns] = true
	}
}

func exploreBaseURL() string {
	return strings.TrimRight(RSSHubURL, "/")
}

func getExploreSnapshot() (*ExploreData, bool, string) {
	exploreMu.RLock()
	defer exploreMu.RUnlock()
	return exploreData, exploreLoading.Load(), exploreLastErr
}

// ensureExploreData 在数据缺失或过期时异步刷新，避免阻塞页面请求
func ensureExploreData() {
	exploreMu.RLock()
	data := exploreData
	exploreMu.RUnlock()

	if data != nil && time.Since(time.Unix(data.UpdatedAt, 0)) < exploreTTL {
		return
	}

	forceRefreshExplore()
}

// forceRefreshExplore 触发一次后台刷新，已有刷新在进行时直接返回
func forceRefreshExplore() {
	if !exploreLoading.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer exploreLoading.Store(false)
		if err := refreshExploreData(); err != nil {
			log.Errorf("refresh rsshub explore data: %v", err)
			exploreMu.Lock()
			exploreLastErr = err.Error()
			exploreMu.Unlock()
		}
	}()
}

func refreshExploreData() error {
	raw, err := fetchRSSHubNamespaces()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	data := buildExploreData(raw)
	exploreMu.Lock()
	exploreData = data
	exploreLastErr = ""
	exploreMu.Unlock()

	record := ExploreCache{Key: exploreCacheKey, Data: string(payload), UpdatedAt: data.UpdatedAt}
	if err := globalDB.Where("key = ?", exploreCacheKey).Assign(record).FirstOrCreate(&ExploreCache{}).Error; err != nil {
		log.Errorf("persist rsshub explore cache: %v", err)
	}

	return nil
}

// loadExploreCache 在启动时把上次抓取的注册表读回内存
func loadExploreCache() {
	var record ExploreCache
	if err := globalDB.Where("key = ?", exploreCacheKey).First(&record).Error; err != nil || record.Data == "" {
		return
	}

	var raw map[string]exploreRawNamespace
	if err := json.Unmarshal([]byte(record.Data), &raw); err != nil {
		log.Errorf("decode rsshub explore cache: %v", err)
		return
	}

	data := buildExploreData(raw)
	if record.UpdatedAt > 0 {
		data.UpdatedAt = record.UpdatedAt
	}
	exploreMu.Lock()
	exploreData = data
	exploreMu.Unlock()
	log.Infof("loaded rsshub explore cache: %d routes, updated at %s", data.RouteCount, time.Unix(data.UpdatedAt, 0).Format(TimeFormat))
}

func fetchRSSHubNamespaces() (map[string]exploreRawNamespace, error) {
	url := exploreBaseURL() + "/api/namespace"
	client := &http.Client{Timeout: 120 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw map[string]exploreRawNamespace
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode %s: %v", url, err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("rsshub %s returned no namespace", exploreBaseURL())
	}

	return raw, nil
}

func buildExploreData(raw map[string]exploreRawNamespace) *ExploreData {
	base := exploreBaseURL()
	items := make([]ExploreItem, 0, 4096)
	seen := make(map[string]bool, 4096)
	categoryCount := make(map[string]int)

	for ns, namespace := range raw {
		nsName := strings.TrimSpace(namespace.Name)
		if nsName == "" {
			nsName = ns
		}

		for path, route := range namespace.Routes {
			example := strings.TrimSpace(route.Example)
			if example == "" || strings.Contains(example, ":") {
				continue
			}
			if !strings.HasPrefix(example, "/") {
				example = "/" + example
			}

			link := base + example
			if seen[link] {
				continue
			}
			seen[link] = true

			name := strings.TrimSpace(route.Name)
			if name == "" {
				name = strings.TrimSpace(path)
			}

			item := ExploreItem{
				Key:           ns + path,
				Namespace:     ns,
				NamespaceName: nsName,
				Name:          name,
				Example:       example,
				Link:          link,
				Description:   plainSummary(route.Description, exploreDescLimit),
				Categories:    route.Categories,
				RequiresSetup: rawFeatureFlag(route.Features.RequireConfig),
				AntiCrawler:   rawFeatureFlag(route.Features.AntiCrawler),
				View:          route.View,
			}
			items = append(items, item)

			for _, cat := range route.Categories {
				categoryCount[cat]++
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].NamespaceName != items[j].NamespaceName {
			return items[i].NamespaceName < items[j].NamespaceName
		}
		return items[i].Name < items[j].Name
	})

	categories := make([]ExploreCategory, 0, len(categoryCount))
	for cat, count := range categoryCount {
		label := exploreCategoryLabels[cat]
		if label == "" {
			label = cat
		}
		categories = append(categories, ExploreCategory{Name: cat, Label: label, Count: count})
	}
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].Count != categories[j].Count {
			return categories[i].Count > categories[j].Count
		}
		return categories[i].Name < categories[j].Name
	})

	return &ExploreData{
		Items:      items,
		Categories: categories,
		RouteCount: len(items),
		UpdatedAt:  time.Now().Unix(),
	}
}

// rawFeatureFlag 兼容 RSSHub 里 bool 与字符串数组两种 feature 写法
func rawFeatureFlag(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}

	var flag bool
	if err := json.Unmarshal(raw, &flag); err == nil {
		return flag
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return len(list) > 0
	}

	return false
}

// plainSummary 把路由说明里的 markdown 噪音去掉，只留可读的一小段
func plainSummary(md string, limit int) string {
	text := md
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "|", " ")
	text = strings.ReplaceAll(text, "`", "")

	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ":::") {
			continue
		}
		kept = append(kept, trimmed)
	}
	text = strings.Join(kept, " ")

	// [text](url) => text
	for {
		open := strings.Index(text, "[")
		if open < 0 {
			break
		}
		closeBracket := strings.Index(text[open:], "](")
		if closeBracket < 0 {
			break
		}
		closeBracket += open
		end := strings.Index(text[closeBracket:], ")")
		if end < 0 {
			break
		}
		end += closeBracket
		text = text[:open] + text[open+1:closeBracket] + text[end+1:]
	}

	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "##", "")
	text = strings.ReplaceAll(text, "#", "")
	text = strings.Join(strings.Fields(text), " ")

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

func filterExploreItems(items []ExploreItem, query, category string) []ExploreItem {
	query = strings.ToLower(strings.TrimSpace(query))
	category = strings.TrimSpace(category)

	filtered := make([]ExploreItem, 0, len(items))
	for _, item := range items {
		if query != "" {
			haystack := strings.ToLower(item.Name + " " + item.Namespace + " " + item.NamespaceName + " " + item.Example + " " + item.Description + " " + strings.Join(item.Categories, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}

		switch category {
		case "", "featured":
			if query == "" && !exploreFeatured[item.Namespace] {
				continue
			}
		case "all":
			// 不加限制
		default:
			if !containsString(item.Categories, category) {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	// 推荐/搜索按热度排，分类内保持按命名空间归组
	if query == "" && (category == "" || category == "featured") {
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].View > filtered[j].View
		})
	}

	return filtered
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// getSubscribedFeedURLs 返回当前用户已订阅的 feed 地址（归一化后）
func getSubscribedFeedURLs(email string) map[string]int64 {
	var feeds []Feed
	globalDB.Select("id", "url").Where("email = ?", email).Find(&feeds)

	result := make(map[string]int64, len(feeds))
	for _, feed := range feeds {
		result[normalizeFeedURL(feed.URL)] = feed.ID
	}
	return result
}

func normalizeFeedURL(url string) string {
	return strings.TrimRight(strings.TrimSpace(url), "/")
}

// ExplorePage 是 explore 模板需要的渲染数据
type ExplorePage struct {
	SiteURL      string
	Items        []ExploreItem
	Categories   []ExploreCategory
	Category     string
	Query        string
	Total        int
	RouteCount   int
	Page         int
	TotalPages   int
	HasPrev      bool
	HasNext      bool
	PrevURL      string
	NextURL      string
	UpdatedAt    int64
	Loading      bool
	ErrorMessage string
	Notice       string
	Back         string
	RSSHubURL    string
	PageSize     int
}

// exploreURL 拼出 explore 页面的地址，默认分类与第一页不写入 query
func exploreURL(query, category string, page int) string {
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if category != "" && category != "featured" {
		values.Set("category", category)
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}

	if len(values) == 0 {
		return "/explore"
	}
	return "/explore?" + values.Encode()
}

func exploreJoin(back string) string {
	if strings.Contains(back, "?") {
		return "&"
	}
	return "?"
}

func paginateExplore(items []ExploreItem, page int) ([]ExploreItem, int, int) {
	totalPages := (len(items) + explorePageSize - 1) / explorePageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * explorePageSize
	end := start + explorePageSize
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}

	return items[start:end], page, totalPages
}
