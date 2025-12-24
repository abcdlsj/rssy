import { NextRequest, NextResponse } from "next/server"
import { prisma } from "@/lib/prisma"
import { generateAISummary } from "@/lib/ai"
import { format, startOfDay, endOfDay, subDays } from "date-fns"

export async function GET(request: NextRequest) {
  const authHeader = request.headers.get("authorization")
  if (authHeader !== `Bearer ${process.env.CRON_SECRET}`) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  try {
    const preferences = await prisma.userPreference.findMany({
      where: {
        enableAISummary: true,
        openAIAPIKey: { not: null },
      },
    })

    const currentTime = format(new Date(), "HH:mm")
    const yesterday = subDays(new Date(), 1)
    const dateStr = format(yesterday, "yyyy-MM-dd")

    let summariesCreated = 0

    for (const pref of preferences) {
      if (pref.aiSummaryTime !== currentTime) continue

      const existingSummary = await prisma.aISummary.findUnique({
        where: {
          userId_date: {
            userId: pref.userId,
            date: dateStr,
          },
        },
      })

      if (existingSummary) continue

      const articles = await prisma.article.findMany({
        where: {
          userId: pref.userId,
          publishAt: {
            gte: startOfDay(yesterday),
            lt: endOfDay(yesterday),
          },
        },
        include: {
          feed: {
            select: { title: true },
          },
        },
      })

      if (articles.length === 0) continue

      const formattedArticles = articles.map((a) => ({
        title: a.title,
        link: a.link,
        content: a.content,
        feedTitle: a.feed.title,
      }))

      const { title, summary } = await generateAISummary(
        formattedArticles,
        pref.openAIAPIKey!,
        pref.openAIEndpoint || undefined,
        pref.aiSummaryPrompt || undefined
      )

      await prisma.aISummary.create({
        data: {
          userId: pref.userId,
          date: dateStr,
          title,
          summary,
          articleCount: articles.length,
        },
      })

      summariesCreated++
    }

    return NextResponse.json({ success: true, summariesCreated })
  } catch (error) {
    console.error("AI Summary cron error:", error)
    return NextResponse.json({ error: "Internal error" }, { status: 500 })
  }
}
