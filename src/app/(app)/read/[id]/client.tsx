"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useArticleReader } from "@/contexts/article-reader-context"

interface ReadPageClientProps {
  article: {
    id: string
    title: string
    link: string
    content: string | null
    feedTitle: string
    publishAt: Date | string
    starred: boolean
  }
}

export function ReadPageClient({ article }: ReadPageClientProps) {
  const router = useRouter()
  const { openArticle } = useArticleReader()

  useEffect(() => {
    openArticle(article)
    router.replace("/")
  }, [article, openArticle, router])

  return null
}
