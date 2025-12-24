import { auth, signIn } from "@/lib/auth"
import { redirect } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Rss } from "lucide-react"

const isDevMode = process.env.DEV_MODE === "true"

export default async function LoginPage() {
  const session = await auth()

  if (session) {
    redirect("/")
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <div className="mb-6 inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-foreground">
            <Rss className="h-7 w-7 text-background" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            RSSy
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            现代化 RSS 阅读器，支持 AI 智能总结
          </p>
        </div>

        <div className="rounded-xl border bg-card p-6 shadow-sm">
          {isDevMode ? (
            <form
              action={async (formData: FormData) => {
                "use server"
                await signIn("credentials", {
                  email: formData.get("email"),
                  redirectTo: "/",
                })
              }}
              className="space-y-4"
            >
              <div className="space-y-2">
                <label
                  htmlFor="email"
                  className="text-sm font-medium text-foreground"
                >
                  邮箱地址
                </label>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  placeholder="your@email.com"
                  defaultValue="dev@local.test"
                  size="lg"
                  required
                />
              </div>
              <Button type="submit" className="w-full" size="lg">
                开发模式登录
              </Button>
              <p className="text-center text-xs text-muted-foreground">
                开发环境下可使用任意邮箱登录
              </p>
            </form>
          ) : (
            <form
              action={async () => {
                "use server"
                await signIn("github", { redirectTo: "/" })
              }}
              className="space-y-4"
            >
              <Button type="submit" className="w-full" size="lg">
                <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                </svg>
                使用 GitHub 登录
              </Button>
              <p className="text-center text-xs text-muted-foreground">
                通过 GitHub 账号安全登录
              </p>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
