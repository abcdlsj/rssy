import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"

export async function GET() {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const summaries = await prisma.aISummary.findMany({
    where: { userId: session.user.id },
    orderBy: { date: "desc" },
    take: 30,
  })

  return NextResponse.json(summaries)
}
