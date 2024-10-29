package internal

import (
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/dustin/go-humanize"
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
	}

	tmpl = template.Must(template.New("").Funcs(tmplFuncs).ParseFS(tmplFS, "tmpl/*.html"))
)
