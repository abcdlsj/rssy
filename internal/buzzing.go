package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var (
	globalBuzzingFeed          = BuzzingFeed{}
	globalBuzzingFeedUpdatedAt = time.Unix(0, 0)
	buzzingFeedLoadFromRemote  = os.Getenv("BUZZING_REMOTE") == "true"
)

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
