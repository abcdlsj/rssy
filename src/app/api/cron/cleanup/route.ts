import { NextRequest, NextResponse } from "next/server"
import { prisma } from "@/lib/prisma"
import { subDays } from "date-fns"

export async function GET(request: NextRequest) {
  const authHeader = request.headers.get("authorization")
  if (authHeader !== `Bearer ${process.env.CRON_SECRET}`) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  try {
    const preferences = await prisma.userPreference.findMany({
      where: { enableAutoCleanup: true },
    })

    let totalDeleted = 0

    for (const pref of preferences) {
      const cutoffDate = subDays(new Date(), pref.cleanupExpiredDays)

      // 只删除已读且过期的文章，保留收藏的
      const result = await prisma.article.deleteMany({
        where: {
          userId: pref.userId,
          read: true,
          starred: false,
          publishAt: { lt: cutoffDate },
        },
      })

      totalDeleted += result.count
    }

    return NextResponse.json({
      success: true,
      articlesDeleted: totalDeleted,
    })
  } catch (error) {
    console.error("Cleanup cron error:", error)
    return NextResponse.json({ error: "Internal error" }, { status: 500 })
  }
}
