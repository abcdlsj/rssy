"use client"

import { useEffect, useRef, useState } from "react"
import { X, ExternalLink, Star, Loader2, ArrowLeft } from "lucide-react"
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
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isOpen) {
      setIsVisible(true)
      document.body.style.overflow = "hidden"
      requestAnimationFrame(() => {
        requestAnimationFrame(() => setIsAnimating(true))
      })
    } else {
      setIsAnimating(false)
      document.body.style.overflow = ""
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
      {/* 桌面端：居中浮窗全覆盖；移动端：底部滑入全屏 */}
      <div
        className={cn(
          "fixed inset-0 z-50 bg-background transition-opacity duration-300",
          "sm:flex sm:items-center sm:justify-center",
          isAnimating ? "opacity-100" : "opacity-0"
        )}
        onClick={closeReader}
      >
        {/* 内容容器 */}
        <article
          className={cn(
            "flex h-full w-full flex-col bg-background transition-transform duration-300 ease-out",
            "sm:h-auto sm:max-h-[90vh] sm:max-w-3xl sm:rounded-xl sm:border sm:shadow-2xl",
            isAnimating
              ? "translate-y-0 sm:scale-100"
              : "translate-y-full sm:translate-y-0 sm:scale-95"
          )}
          onClick={(e) => e.stopPropagation()}
        >
          {selectedArticle && (
            <>
              {/* Header */}
              <header className="flex shrink-0 items-center justify-between border-b bg-background px-4 py-3 sm:rounded-t-xl sm:px-6">
                {/* 移动端显示返回按钮 */}
                <button
                  onClick={closeReader}
                  className="flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground sm:hidden"
                >
                  <ArrowLeft className="h-4 w-4" />
                  <span>返回</span>
                </button>

                {/* 桌面端显示 feed 信息 */}
                <div className="hidden items-center gap-2 text-sm text-muted-foreground sm:flex">
                  <span className="max-w-[200px] truncate">{selectedArticle.feedTitle}</span>
                  <span className="opacity-50">·</span>
                  <time>{timeAgo(selectedArticle.publishAt)}</time>
                </div>

                {/* 操作按钮 */}
                <div className="flex items-center gap-0.5">
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
                    title="查看原文"
                  >
                    <ExternalLink className="h-5 w-5" />
                  </a>
                  {/* 桌面端显示关闭按钮 */}
                  <button
                    onClick={closeReader}
                    className="hidden rounded-lg p-2 text-muted-foreground transition-colors hover:text-foreground sm:block"
                    title="关闭 (Esc)"
                  >
                    <X className="h-5 w-5" />
                  </button>
                </div>
              </header>

              {/* Content */}
              <div ref={contentRef} className="flex-1 overflow-y-auto">
                <div className="mx-auto max-w-2xl px-4 py-6 sm:px-6 sm:py-8">
                  {/* 移动端显示 feed 信息 */}
                  <div className="mb-3 flex items-center gap-2 text-sm text-muted-foreground sm:hidden">
                    <span className="truncate">{selectedArticle.feedTitle}</span>
                    <span className="opacity-50">·</span>
                    <time>{timeAgo(selectedArticle.publishAt)}</time>
                  </div>

                  {/* 标题 */}
                  <h1 className="mb-6 text-xl font-semibold leading-tight tracking-tight sm:text-2xl lg:text-[1.75rem]">
                    {selectedArticle.title}
                  </h1>

                  {/* 文章内容 */}
                  {loading ? (
                    <div className="flex items-center justify-center py-16">
                      <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                  ) : (fullArticle?.fullContent || fullArticle?.content || selectedArticle.content) ? (
                    <div
                      className="prose prose-neutral dark:prose-invert max-w-none prose-headings:font-semibold prose-headings:tracking-tight prose-p:leading-relaxed prose-a:text-primary prose-a:no-underline hover:prose-a:underline prose-img:rounded-lg prose-img:shadow-md"
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
            </>
          )}
        </article>
      </div>
    </>
  )
}
