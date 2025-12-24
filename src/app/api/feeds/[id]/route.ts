import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"

export async function GET(
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
    include: {
      articles: {
        orderBy: { publishAt: "desc" },
      },
    },
  })

  if (!feed) {
    return NextResponse.json({ error: "Feed not found" }, { status: 404 })
  }

  return NextResponse.json(feed)
}

export async function DELETE(
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

  await prisma.feed.delete({ where: { id: feedId } })

  return NextResponse.json({ success: true })
}
