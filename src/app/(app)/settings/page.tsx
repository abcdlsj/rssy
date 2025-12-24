import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { SettingsForm } from "@/components/settings-form"

export default async function SettingsPage() {
  const session = await auth()
  if (!session?.user?.id) return null

  let preference = await prisma.userPreference.findUnique({
    where: { userId: session.user.id },
  })

  if (!preference) {
    preference = await prisma.userPreference.create({
      data: { userId: session.user.id },
    })
  }

  const safePreference = {
    cleanupExpiredDays: preference.cleanupExpiredDays,
    enableAutoCleanup: preference.enableAutoCleanup,
    enableAISummary: preference.enableAISummary,
    aiSummaryTime: preference.aiSummaryTime,
    aiSummaryPrompt: preference.aiSummaryPrompt,
    openAIAPIKey: preference.openAIAPIKey ? "********" : "",
    openAIEndpoint: preference.openAIEndpoint,
  }

  return (
    <div>
      <header className="mb-8">
        <h1 className="text-lg font-medium text-foreground">设置</h1>
      </header>
      <SettingsForm preference={safePreference} />
    </div>
  )
}
