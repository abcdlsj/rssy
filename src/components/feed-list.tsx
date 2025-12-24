"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { timeAgo } from "@/lib/utils"
import { RefreshCw, Trash2, Loader2, ExternalLink, Rss } from "lucide-react"

interface Feed {
  id: number
  url: string
  title: string
  lastFetchedAt: Date | null
  createdAt: Date
  _count: {
    articles: number
  }
}

export function FeedList({ feeds: initialFeeds }: { feeds: Feed[] }) {
  const [feeds, setFeeds] = useState(initialFeeds)
  const [refreshing, setRefreshing] = useState<number | null>(null)
  const router = useRouter()

  const refreshFeed = async (id: number) => {
    setRefreshing(id)
    try {
      await fetch(`/api/feeds/${id}/refresh`, { method: "POST" })
      router.refresh()
    } finally {
      setRefreshing(null)
    }
  }

  const deleteFeed = async (id: number) => {
    if (!confirm("确定删除？")) return
    await fetch(`/api/feeds/${id}`, { method: "DELETE" })
    setFeeds((prev) => prev.filter((f) => f.id !== id))
  }

  if (feeds.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <Rss className="mb-3 h-8 w-8 text-muted-foreground/30" />
        <p className="text-sm text-muted-foreground">添加第一个订阅源</p>
      </div>
    )
  }

  return (
    <div className="space-y-px">
      {feeds.map((feed) => (
        <div
          key={feed.id}
          className="group -mx-3 flex items-center justify-between rounded-lg px-3 py-2.5 transition-colors hover:bg-muted/50"
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <Link
                href={`/feeds/${feed.id}`}
                className="truncate text-[15px] text-foreground hover:underline hover:underline-offset-2"
              >
                {feed.title}
              </Link>
              {feed._count.articles > 0 && (
                <span className="shrink-0 text-xs text-muted-foreground">
                  {feed._count.articles}
                </span>
              )}
            </div>
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {feed.lastFetchedAt ? timeAgo(feed.lastFetchedAt) : feed.url}
            </p>
          </div>

          <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
            <button
              onClick={() => refreshFeed(feed.id)}
              disabled={refreshing === feed.id}
              className="rounded p-1.5 text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
              title="刷新"
            >
              {refreshing === feed.id ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
            </button>
            <a
              href={feed.url}
              target="_blank"
              rel="noopener noreferrer"
              className="rounded p-1.5 text-muted-foreground transition-colors hover:text-foreground"
              title="源地址"
            >
              <ExternalLink className="h-4 w-4" />
            </a>
            <button
              onClick={() => deleteFeed(feed.id)}
              className="rounded p-1.5 text-muted-foreground transition-colors hover:text-destructive"
              title="删除"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}
