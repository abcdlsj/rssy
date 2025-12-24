import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { parseFeed, filterRecentItems } from "@/lib/rss"

export async function POST(
  _request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const { id } = await params
  const feedId = parseInt(id)

  const feed = await prisma.feed.findFirst({
    where: { id: feedId, userId: session.user.id },
  })

  if (!feed) {
    return NextResponse.json({ error: "Feed not found" }, { status: 404 })
  }

  try {
    const parsed = await parseFeed(feed.url)
    const recentItems = filterRecentItems(parsed.items, 7)

    const existingTitles = await prisma.article.findMany({
      where: { feedId: feed.id },
      select: { title: true },
    })
    const titleSet = new Set(existingTitles.map((a) => a.title))

    const newArticles = recentItems
      .filter((item) => !titleSet.has(item.title))
      .map((item) => ({
        feedId: feed.id,
        userId: session.user.id,
        title: item.title,
        link: item.link,
        content: item.content || null,
        publishAt: item.pubDate ? new Date(item.pubDate) : new Date(),
      }))

    if (newArticles.length > 0) {
      await prisma.article.createMany({ data: newArticles })
    }

    await prisma.feed.update({
      where: { id: feedId },
      data: { lastFetchedAt: new Date() },
    })

    return NextResponse.json({ articlesAdded: newArticles.length })
  } catch (error) {
    console.error("Error refreshing feed:", error)
    return NextResponse.json({ error: "Failed to refresh feed" }, { status: 500 })
  }
}
