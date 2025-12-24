import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"

export async function GET() {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  let preference = await prisma.userPreference.findUnique({
    where: { userId: session.user.id },
  })

  if (!preference) {
    preference = await prisma.userPreference.create({
      data: { userId: session.user.id },
    })
  }

  return NextResponse.json({
    ...preference,
    openAIAPIKey: preference.openAIAPIKey ? "********" : null,
  })
}

export async function PATCH(request: NextRequest) {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const body = await request.json()
  const updateData: Record<string, unknown> = {}

  if (body.cleanupExpiredDays !== undefined) updateData.cleanupExpiredDays = body.cleanupExpiredDays
  if (body.enableAutoCleanup !== undefined) updateData.enableAutoCleanup = body.enableAutoCleanup
  if (body.aiSummaryPrompt !== undefined) updateData.aiSummaryPrompt = body.aiSummaryPrompt
  if (body.enableAISummary !== undefined) updateData.enableAISummary = body.enableAISummary
  if (body.aiSummaryTime !== undefined) updateData.aiSummaryTime = body.aiSummaryTime
  if (body.openAIAPIKey !== undefined && body.openAIAPIKey !== "********") {
    updateData.openAIAPIKey = body.openAIAPIKey
  }
  if (body.openAIEndpoint !== undefined) updateData.openAIEndpoint = body.openAIEndpoint

  const preference = await prisma.userPreference.upsert({
    where: { userId: session.user.id },
    update: updateData,
    create: {
      userId: session.user.id,
      ...updateData,
    },
  })

  return NextResponse.json({
    ...preference,
    openAIAPIKey: preference.openAIAPIKey ? "********" : null,
  })
}
