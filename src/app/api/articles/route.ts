import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"

export async function GET(request: NextRequest) {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const searchParams = request.nextUrl.searchParams
  const feedId = searchParams.get("feedId")
  const unreadOnly = searchParams.get("unread") !== "false"

  const articles = await prisma.article.findMany({
    where: {
      userId: session.user.id,
      ...(unreadOnly && { read: false }),
      ...(feedId && { feedId: parseInt(feedId) }),
    },
    orderBy: { publishAt: "desc" },
    include: {
      feed: {
        select: { title: true },
      },
    },
  })

  return NextResponse.json(articles)
}
