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

  const article = await prisma.article.findFirst({
    where: { id, userId: session.user.id },
    include: {
      feed: {
        select: { title: true },
      },
    },
  })

  if (!article) {
    return NextResponse.json({ error: "Article not found" }, { status: 404 })
  }

  return NextResponse.json(article)
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const { id } = await params
  const body = await request.json()

  const article = await prisma.article.findFirst({
    where: { id, userId: session.user.id },
  })

  if (!article) {
    return NextResponse.json({ error: "Article not found" }, { status: 404 })
  }

  const updated = await prisma.article.update({
    where: { id },
    data: {
      read: body.read ?? article.read,
      starred: body.starred ?? article.starred,
    },
  })

  return NextResponse.json(updated)
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

  const article = await prisma.article.findFirst({
    where: { id, userId: session.user.id },
  })

  if (!article) {
    return NextResponse.json({ error: "Article not found" }, { status: 404 })
  }

  await prisma.article.delete({
    where: { id },
  })

  return NextResponse.json({ success: true })
}
