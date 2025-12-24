"use client"

import { useEffect, useRef, useState } from "react"
import { X, ExternalLink, Star, Loader2 } from "lucide-react"
import { useArticleReader } from "@/contexts/article-reader-context"
import { timeAgo, cn } from "@/lib/utils"

interface FullArticle {
  id: string
  title: string
  link: string
  content: string | null
  fullContent: string | null
  feedTitle: string
  publishAt: Date | string
  starred: boolean
  feed: { title: string }
}

export function ArticleSidebar() {
  const { selectedArticle, isOpen, closeReader } = useArticleReader()
  const [starred, setStarred] = useState(false)
  const [fullArticle, setFullArticle] = useState<FullArticle | null>(null)
  const [loading, setLoading] = useState(false)
  const [isVisible, setIsVisible] = useState(false)
  const [isAnimating, setIsAnimating] = useState(false)
  const sidebarRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isOpen) {
      setIsVisible(true)
      requestAnimationFrame(() => {
        requestAnimationFrame(() => setIsAnimating(true))
      })
    } else {
      setIsAnimating(false)
      const timer = setTimeout(() => setIsVisible(false), 300)
      return () => clearTimeout(timer)
    }
  }, [isOpen])

  useEffect(() => {
    if (selectedArticle) {
      setStarred(selectedArticle.starred)
      setFullArticle(null)
      setLoading(true)

      fetch(`/api/articles/${selectedArticle.id}`)
        .then((res) => res.json())
        .then((data) => {
          setFullArticle(data)
          setLoading(false)
        })
        .catch(() => setLoading(false))

      fetch(`/api/articles/${selectedArticle.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ read: true }),
      })
    }
  }, [selectedArticle])

  useEffect(() => {
    if (contentRef.current) {
      contentRef.current.scrollTop = 0
    }
  }, [selectedArticle?.id])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        closeReader()
      }
    }
    document.addEventListener("keydown", handleEscape)
    return () => document.removeEventListener("keydown", handleEscape)
  }, [isOpen, closeReader])

  const toggleStar = async () => {
    if (!selectedArticle) return
    await fetch(`/api/articles/${selectedArticle.id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ starred: !starred }),
    })
    setStarred(!starred)
  }

  if (!isVisible) return null

  return (
    <>
      {/* Backdrop - 透明点击区域 */}
      <div
        className="fixed inset-0 z-40"
        onClick={closeReader}
      />

      {/* Sidebar */}
      <aside
        ref={sidebarRef}
        className={cn(
          "fixed right-0 top-14 z-50 h-[calc(100vh-3.5rem)] w-full border-l bg-background shadow-2xl transition-transform duration-300 ease-out sm:w-[640px] lg:w-[720px]",
          isAnimating ? "translate-x-0" : "translate-x-full"
        )}
      >
        {selectedArticle && (
          <div className="flex h-full flex-col">
            {/* Header */}
            <header className="flex items-center justify-between border-b px-4 py-3 lg:px-6">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="truncate">{selectedArticle.feedTitle}</span>
                <span>·</span>
                <time>{timeAgo(selectedArticle.publishAt)}</time>
              </div>

              <div className="flex items-center gap-1">
                <button
                  onClick={toggleStar}
                  className={cn(
                    "rounded-lg p-2 transition-colors",
                    starred
                      ? "text-amber-500"
                      : "text-muted-foreground hover:text-amber-500"
                  )}
                  title={starred ? "取消收藏" : "收藏"}
                >
                  <Star className={cn("h-5 w-5", starred && "fill-current")} />
                </button>
                <a
                  href={selectedArticle.link}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="rounded-lg p-2 text-muted-foreground transition-colors hover:text-foreground"
                  title="原文"
                >
                  <ExternalLink className="h-5 w-5" />
                </a>
                <button
                  onClick={closeReader}
                  className="rounded-lg p-2 text-muted-foreground transition-colors hover:text-foreground"
                  title="关闭"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            </header>

            {/* Content */}
            <div ref={contentRef} className="flex-1 overflow-y-auto px-4 py-6 lg:px-6">
              <h1 className="mb-6 text-xl font-semibold leading-tight tracking-tight lg:text-2xl">
                {selectedArticle.title}
              </h1>

              {loading ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : (fullArticle?.fullContent || fullArticle?.content || selectedArticle.content) ? (
                <div
                  className="prose prose-neutral dark:prose-invert max-w-none prose-headings:font-semibold prose-headings:tracking-tight prose-a:text-primary prose-a:no-underline hover:prose-a:underline prose-img:rounded-lg"
                  dangerouslySetInnerHTML={{
                    __html: fullArticle?.fullContent || fullArticle?.content || selectedArticle.content || "",
                  }}
                />
              ) : (
                <div className="rounded-lg border border-dashed p-8 text-center">
                  <p className="text-muted-foreground">
                    无法获取文章内容，请
                    <a
                      href={selectedArticle.link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="mx-1 text-primary hover:underline"
                    >
                      查看原文
                    </a>
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
      </aside>
    </>
  )
}
