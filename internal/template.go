package internal

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/russross/blackfriday/v2"
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

		"markdownToHTML": func(content string) template.HTML {
			renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
				Flags: blackfriday.CommonHTMLFlags | blackfriday.HrefTargetBlank | blackfriday.SkipHTML,
			})

			extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs

			html := blackfriday.Run([]byte(content), blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))
			return template.HTML(html)
		},

		"summaryNeedsPreview": func(content string) bool {
			return len([]rune(content)) > 2400
		},

		"summaryPreview": func(content string) string {
			const limit = 2400
			runes := []rune(strings.TrimSpace(content))
			if len(runes) <= limit {
				return content
			}

			preview := string(runes[:limit-1])
			if paragraphEnd := strings.LastIndex(preview, "\n\n"); paragraphEnd > limit*2/3 {
				preview = preview[:paragraphEnd]
			}
			return strings.TrimSpace(preview) + "…"
		},

		"displayContentRead": func(content string) bool {
			return len(content) >= 30
		},

		"buzTimeformat": func(t string) string {
			tm, err := time.Parse(time.RFC3339Nano, t)
			if err != nil {
				return t
			}
			return humanize.Time(tm)
		},
		"enableReadabilityButton": func(feedID int64) bool {
			return getFeedMetaWithCache(feedID).EnableReadability
		},

		"getFeedHighlight": func(feedID int64) bool {
			return getFeedMetaWithCache(feedID).Highlight
		},

		"getFeedHideUnread": func(feedID int64) bool {
			return getFeedMetaWithCache(feedID).HideUnread
		},

		"getFeedCategory": func(feedID int64) string {
			return getFeedMetaWithCache(feedID).Categories
		},

		"splitLines": func(text string) []string {
			return strings.Split(text, "\n")
		},

		"buildReadabilityURL": func(articleLink string) string {
			template := ReadabilityURLTemplate
			if template == "" {
				template = "https://reada.songjian.li/read/https://r.jina.ai/%s?md=true&nocache=true"
			}
			return fmt.Sprintf(template, articleLink)
		},

		"getTimeCategory": func(publishAt int64) string {
			t := time.Unix(publishAt, 0)
			now := time.Now()
			diff := now.Sub(t)
			days := int(diff.Hours() / 24)

			if days < 3 {
				return "recent"
			} else if days < 7 {
				return "week"
			} else {
				return "older"
			}
		},
	}

	tmpl = template.Must(template.New("").Funcs(tmplFuncs).ParseFS(tmplFS, "tmpl/*.html"))
)
