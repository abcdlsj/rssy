import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { ArticleList } from "@/components/article-list"
import Link from "next/link"

export default async function HomePage({
  searchParams,
}: {
  searchParams: Promise<{ view?: string }>
}) {
  const session = await auth()
  if (!session?.user?.id) return null

  const { view } = await searchParams
  const isStarred = view === "starred"

  const articles = await prisma.article.findMany({
    where: {
      userId: session.user.id,
      ...(isStarred
        ? { starred: true }
        : { read: false }),
    },
    orderBy: { publishAt: "desc" },
    include: {
      feed: {
        select: { title: true },
      },
    },
    take: 100,
  })

  return (
    <div>
      <header className="mb-6 flex items-center gap-4">
        <Link
          href="/"
          className={`text-lg font-medium ${!isStarred ? "text-foreground" : "text-muted-foreground hover:text-foreground"}`}
        >
          未读
          {!isStarred && (
            <span className="ml-1.5 text-muted-foreground">{articles.length}</span>
          )}
        </Link>
        <Link
          href="/?view=starred"
          className={`text-lg font-medium ${isStarred ? "text-foreground" : "text-muted-foreground hover:text-foreground"}`}
        >
          收藏
          {isStarred && (
            <span className="ml-1.5 text-muted-foreground">{articles.length}</span>
          )}
        </Link>
      </header>

      <ArticleList articles={articles} />
    </div>
  )
}
