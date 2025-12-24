"use client"

import { createContext, useContext, useState, useCallback, ReactNode } from "react"

interface Article {
  id: string
  title: string
  link: string
  content: string | null
  feedTitle: string
  publishAt: Date | string
  starred: boolean
}

interface ArticleReaderContextType {
  selectedArticle: Article | null
  isOpen: boolean
  openArticle: (article: Article) => void
  closeReader: () => void
}

const ArticleReaderContext = createContext<ArticleReaderContextType | undefined>(undefined)

export function ArticleReaderProvider({ children }: { children: ReactNode }) {
  const [selectedArticle, setSelectedArticle] = useState<Article | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  const openArticle = useCallback((article: Article) => {
    setSelectedArticle(article)
    setIsOpen(true)
  }, [])

  const closeReader = useCallback(() => {
    setIsOpen(false)
    setTimeout(() => setSelectedArticle(null), 300)
  }, [])

  return (
    <ArticleReaderContext.Provider
      value={{ selectedArticle, isOpen, openArticle, closeReader }}
    >
      {children}
    </ArticleReaderContext.Provider>
  )
}

export function useArticleReader() {
  const context = useContext(ArticleReaderContext)
  if (context === undefined) {
    throw new Error("useArticleReader must be used within an ArticleReaderProvider")
  }
  return context
}
