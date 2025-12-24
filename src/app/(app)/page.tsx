import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { ArticleList } from "@/components/article-list"
import Link from "next/link"

export const dynamic = "force-dynamic"

export default async function HomePage({
  searchParams,
}: {
  searchParams: Promise<{ view?: string }>
}) {
  const session = await auth()
  if (!session?.user?.id) return null

  const { view } = await searchParams
  const currentView = view === "starred" ? "starred" : view === "archive" ? "archive" : "unread"

  const articles = await prisma.article.findMany({
    where: {
      userId: session.user.id,
      ...(currentView === "starred"
        ? { starred: true }
        : currentView === "archive"
          ? { read: true }
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
          className={`text-lg font-medium ${currentView === "unread" ? "text-foreground" : "text-muted-foreground hover:text-foreground"}`}
        >
          未读
          {currentView === "unread" && (
            <span className="ml-1.5 text-muted-foreground">{articles.length}</span>
          )}
        </Link>
        <Link
          href="/?view=starred"
          className={`text-lg font-medium ${currentView === "starred" ? "text-foreground" : "text-muted-foreground hover:text-foreground"}`}
        >
          收藏
          {currentView === "starred" && (
            <span className="ml-1.5 text-muted-foreground">{articles.length}</span>
          )}
        </Link>
        <Link
          href="/?view=archive"
          className={`text-lg font-medium ${currentView === "archive" ? "text-foreground" : "text-muted-foreground hover:text-foreground"}`}
        >
          已读
          {currentView === "archive" && (
            <span className="ml-1.5 text-muted-foreground">{articles.length}</span>
          )}
        </Link>
      </header>

      <ArticleList key={currentView} articles={articles} view={currentView} />
    </div>
  )
}
