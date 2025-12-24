"use client"

import { useState } from "react"
import { timeAgo, cn } from "@/lib/utils"
import { Check, Trash2, ExternalLink, Star, Inbox } from "lucide-react"
import { format, isToday, isYesterday, parseISO } from "date-fns"
import { zhCN } from "date-fns/locale"

interface Article {
  id: string
  title: string
  link: string
  content: string | null
  read: boolean
  starred: boolean
  publishAt: Date | string
  feed: {
    title: string
  }
}

function groupByDate(articles: Article[]) {
  const groups: Record<string, Article[]> = {}

  for (const article of articles) {
    const date = typeof article.publishAt === "string"
      ? parseISO(article.publishAt)
      : article.publishAt
    const key = format(date, "yyyy-MM-dd")
    if (!groups[key]) groups[key] = []
    groups[key].push(article)
  }

  return Object.entries(groups).sort(([a], [b]) => b.localeCompare(a))
}

function formatDateLabel(dateStr: string) {
  const date = parseISO(dateStr)
  if (isToday(date)) return "今天"
  if (isYesterday(date)) return "昨天"
  return format(date, "M月d日 EEEE", { locale: zhCN })
}

export function ArticleList({ articles: initialArticles }: { articles: Article[] }) {
  const [articles, setArticles] = useState(initialArticles)

  const markAsRead = async (id: string) => {
    await fetch(`/api/articles/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ read: true }),
    })
    setArticles((prev) => prev.filter((a) => a.id !== id))
  }

  const toggleStar = async (id: string, starred: boolean) => {
    await fetch(`/api/articles/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ starred: !starred }),
    })
    setArticles((prev) =>
      prev.map((a) => (a.id === id ? { ...a, starred: !starred } : a))
    )
  }

  const deleteArticle = async (id: string) => {
    await fetch(`/api/articles/${id}`, {
      method: "DELETE",
    })
    setArticles((prev) => prev.filter((a) => a.id !== id))
  }

  if (articles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <Inbox className="mb-3 h-8 w-8 text-muted-foreground/30" />
        <p className="text-sm text-muted-foreground">已全部阅读</p>
      </div>
    )
  }

  const grouped = groupByDate(articles)

  return (
    <div className="space-y-8">
      {grouped.map(([dateStr, items]) => (
        <section key={dateStr}>
          <h2 className="mb-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            {formatDateLabel(dateStr)}
          </h2>
          <div className="space-y-px">
            {items.map((article) => (
              <article
                key={article.id}
                className="group -mx-3 flex items-start gap-3 rounded-lg px-3 py-2.5 transition-colors hover:bg-muted/50"
              >
                <div className="min-w-0 flex-1">
                  <div className="mb-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                    <span>{article.feed.title}</span>
                    <span className="opacity-50">·</span>
                    <time>{timeAgo(article.publishAt)}</time>
                  </div>

                  <h3 className="text-[15px] leading-snug text-foreground">
                    <a
                      href={article.link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:underline hover:underline-offset-2"
                      onClick={() => markAsRead(article.id)}
                    >
                      {article.title}
                    </a>
                  </h3>
                </div>

                <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                  <button
                    onClick={() => toggleStar(article.id, article.starred)}
                    className={cn(
                      "rounded p-1.5 transition-colors",
                      article.starred
                        ? "text-amber-500"
                        : "text-muted-foreground hover:text-amber-500"
                    )}
                    title="收藏"
                  >
                    <Star className={cn("h-4 w-4", article.starred && "fill-current")} />
                  </button>
                  <button
                    onClick={() => markAsRead(article.id)}
                    className="rounded p-1.5 text-muted-foreground transition-colors hover:text-foreground"
                    title="已读"
                  >
                    <Check className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => deleteArticle(article.id)}
                    className="rounded p-1.5 text-muted-foreground transition-colors hover:text-destructive"
                    title="删除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                  <a
                    href={article.link}
                    target="_blank"
                    rel="noopener noreferrer"
                    onClick={() => markAsRead(article.id)}
                    className="rounded p-1.5 text-muted-foreground transition-colors hover:text-foreground"
                    title="打开"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </a>
                </div>
              </article>
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}
