import { extract } from "@extractus/article-extractor"

export interface ExtractedArticle {
  title?: string
  content?: string
  author?: string
  published?: string
}

export async function extractArticle(url: string): Promise<ExtractedArticle | null> {
  try {
    const article = await extract(url)

    if (!article) return null

    return {
      title: article.title,
      content: article.content,
      author: article.author,
      published: article.published,
    }
  } catch (error) {
    console.error(`Failed to extract article from ${url}:`, error)
    return null
  }
}
