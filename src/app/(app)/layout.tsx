import { auth } from "@/lib/auth"
import { redirect } from "next/navigation"
import { Nav } from "@/components/nav"
import { SessionProvider } from "next-auth/react"

export default async function AppLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const session = await auth()

  if (!session) {
    redirect("/login")
  }

  return (
    <SessionProvider session={session}>
      <div className="min-h-screen bg-background">
        <Nav />
        <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 sm:py-10">
          {children}
        </main>
      </div>
    </SessionProvider>
  )
}
