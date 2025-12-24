import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { FeedList } from "@/components/feed-list"
import { AddFeedForm } from "@/components/add-feed-form"
import { OPMLButtons } from "@/components/opml-buttons"

export default async function FeedsPage() {
  const session = await auth()
  if (!session?.user?.id) return null

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

  return (
    <div>
      <header className="mb-6 flex items-center justify-between">
        <h1 className="text-lg font-medium text-foreground">
          订阅
          <span className="ml-1.5 text-muted-foreground">{feeds.length}</span>
        </h1>
        <OPMLButtons />
      </header>

      <div className="space-y-6">
        <AddFeedForm />
        <FeedList feeds={feeds} />
      </div>
    </div>
  )
}
