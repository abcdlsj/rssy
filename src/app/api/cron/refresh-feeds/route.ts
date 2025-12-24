import { NextRequest, NextResponse } from "next/server"
import { prisma } from "@/lib/prisma"
import { parseFeed, filterRecentItems } from "@/lib/rss"
import { extractArticle } from "@/lib/extractor"

export async function GET(request: NextRequest) {
  const authHeader = request.headers.get("authorization")
  if (authHeader !== `Bearer ${process.env.CRON_SECRET}`) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  try {
    const feeds = await prisma.feed.findMany({
      where: {
        OR: [
          { lastFetchedAt: null },
          { lastFetchedAt: { lt: new Date(Date.now() - 30 * 60 * 1000) } },
        ],
      },
    })

    let totalArticlesAdded = 0

    for (const feed of feeds) {
      try {
        const parsed = await parseFeed(feed.url)
        const recentItems = filterRecentItems(parsed.items, 7)

        const existingTitles = await prisma.article.findMany({
          where: { feedId: feed.id },
          select: { title: true },
        })
        const titleSet = new Set(existingTitles.map((a) => a.title))

        const newItems = recentItems.filter((item) => !titleSet.has(item.title))

        for (const item of newItems) {
          let fullContent: string | null = null
          try {
            const extracted = await extractArticle(item.link)
            if (extracted?.content) {
              fullContent = extracted.content
            }
          } catch {
            // ignore extraction errors
          }

          await prisma.article.create({
            data: {
              feedId: feed.id,
              userId: feed.userId,
              title: item.title,
              link: item.link,
              content: item.content || null,
              fullContent,
              publishAt: item.pubDate ? new Date(item.pubDate) : new Date(),
            },
          })
          totalArticlesAdded++
        }

        await prisma.feed.update({
          where: { id: feed.id },
          data: { lastFetchedAt: new Date() },
        })
      } catch (error) {
        console.error(`Error refreshing feed ${feed.url}:`, error)
      }
    }

    return NextResponse.json({
      success: true,
      feedsProcessed: feeds.length,
      articlesAdded: totalArticlesAdded,
    })
  } catch (error) {
    console.error("Cron job error:", error)
    return NextResponse.json({ error: "Internal error" }, { status: 500 })
  }
}
