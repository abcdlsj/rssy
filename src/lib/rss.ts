import Parser from "rss-parser"

const parser = new Parser({
  timeout: 10000,
  headers: {
    "User-Agent": "RSSy/1.0",
  },
})

export interface ParsedFeed {
  title: string
  link?: string
  description?: string
  items: ParsedItem[]
}

export interface ParsedItem {
  title: string
  link: string
  content?: string
  pubDate?: string
  creator?: string
}

export async function parseFeed(url: string): Promise<ParsedFeed> {
  const feed = await parser.parseURL(url)

  return {
    title: feed.title || url,
    link: feed.link,
    description: feed.description,
    items: (feed.items || []).map((item) => ({
      title: item.title || "Untitled",
      link: item.link || "",
      content: item.content || item.contentSnippet || "",
      pubDate: item.pubDate || item.isoDate,
      creator: item.creator,
    })),
  }
}

export function filterRecentItems(items: ParsedItem[], days: number = 7): ParsedItem[] {
  const cutoff = Date.now() - days * 24 * 60 * 60 * 1000
  return items.filter((item) => {
    if (!item.pubDate) return false
    const pubTime = new Date(item.pubDate).getTime()
    return pubTime > cutoff
  })
}
