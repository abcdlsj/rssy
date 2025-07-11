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
			// Configure blackfriday to render nice HTML
			renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
				Flags: blackfriday.CommonHTMLFlags | blackfriday.HrefTargetBlank,
			})
			
			extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs
			
			html := blackfriday.Run([]byte(content), blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))
			return template.HTML(html)
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

		"splitLines": func(text string) []string {
			return strings.Split(text, "\n")
		},
	}

	tmpl = template.Must(template.New("").Funcs(tmplFuncs).ParseFS(tmplFS, "tmpl/*.html"))
)
