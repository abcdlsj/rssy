import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { parseFeed, filterRecentItems } from "@/lib/rss"

export async function GET() {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const feeds = await prisma.feed.findMany({
    where: { userId: session.user.id },
    orderBy: { createdAt: "desc" },
    include: {
      _count: {
        select: {
          articles: {
            where: { read: false },
          },
        },
      },
    },
  })

  return NextResponse.json(feeds)
}

export async function POST(request: NextRequest) {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const { url } = await request.json()
  if (!url) {
    return NextResponse.json({ error: "URL is required" }, { status: 400 })
  }

  try {
    const parsed = await parseFeed(url)
    const recentItems = filterRecentItems(parsed.items, 7)

    const feed = await prisma.feed.upsert({
      where: {
        url_userId: {
          url,
          userId: session.user.id,
        },
      },
      update: {
        title: parsed.title,
        lastFetchedAt: new Date(),
      },
      create: {
        url,
        title: parsed.title,
        userId: session.user.id,
        lastFetchedAt: new Date(),
      },
    })

    const existingTitles = await prisma.article.findMany({
      where: { feedId: feed.id, userId: session.user.id },
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

    return NextResponse.json({ feed, articlesAdded: newArticles.length })
  } catch (error) {
    console.error("Error adding feed:", error)
    return NextResponse.json({ error: "Failed to parse feed" }, { status: 400 })
  }
}
