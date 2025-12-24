import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { AISummaryList } from "@/components/ai-summary-list"

export default async function AISummaryPage() {
  const session = await auth()
  if (!session?.user?.id) return null

  const summaries = await prisma.aISummary.findMany({
    where: { userId: session.user.id },
    orderBy: { date: "desc" },
    take: 30,
  })

  return (
    <div>
      <header className="mb-6">
        <h1 className="text-lg font-medium text-foreground">
          AI
          <span className="ml-1.5 text-muted-foreground">{summaries.length}</span>
        </h1>
      </header>
      <AISummaryList summaries={summaries} />
    </div>
  )
}
