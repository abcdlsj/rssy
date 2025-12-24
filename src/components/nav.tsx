"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { Rss, FileText, Sparkles, Settings, LogOut } from "lucide-react"
import { signOut } from "next-auth/react"

const navItems = [
  { href: "/", label: "文章", icon: FileText },
  { href: "/feeds", label: "订阅", icon: Rss },
  { href: "/ai-summary", label: "AI", icon: Sparkles },
  { href: "/settings", label: "设置", icon: Settings },
]

export function Nav() {
  const pathname = usePathname()

  return (
    <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4 sm:px-6">
        <Link
          href="/"
          className="flex items-center gap-2 text-foreground transition-opacity hover:opacity-70"
        >
          <Rss className="h-5 w-5" />
          <span className="font-semibold">RSSy</span>
        </Link>

        <nav className="flex items-center">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive = pathname === item.href
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "relative px-3 py-2 text-sm transition-colors",
                  isActive
                    ? "text-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                <span className="flex items-center gap-1.5">
                  <Icon className="h-4 w-4" />
                  <span className="hidden sm:inline">{item.label}</span>
                </span>
                {isActive && (
                  <span className="absolute inset-x-1 -bottom-[calc(0.5rem+1px)] h-px bg-foreground" />
                )}
              </Link>
            )
          })}

          <button
            onClick={() => signOut()}
            className="ml-2 px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </nav>
      </div>
    </header>
  )
}
