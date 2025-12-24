import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { notFound, redirect } from "next/navigation"
import { ReadPageClient } from "./client"

export default async function ReadPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const session = await auth()
  if (!session?.user?.id) return null

  const { id } = await params

  const article = await prisma.article.findFirst({
    where: {
      id,
      userId: session.user.id,
    },
    include: {
      feed: {
        select: { title: true },
      },
    },
  })

  if (!article) {
    notFound()
  }

  if (!article.read) {
    await prisma.article.update({
      where: { id },
      data: { read: true },
    })
  }

  return (
    <ReadPageClient
      article={{
        id: article.id,
        title: article.title,
        link: article.link,
        content: article.fullContent || article.content,
        feedTitle: article.feed.title,
        publishAt: article.publishAt,
        starred: article.starred,
      }}
    />
  )
}
