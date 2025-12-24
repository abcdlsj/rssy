"use client"

import { useState } from "react"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import { ChevronDown, ChevronUp, Sparkles } from "lucide-react"

interface AISummary {
  id: number
  date: string
  title: string
  summary: string
  articleCount: number
}

export function AISummaryList({ summaries }: { summaries: AISummary[] }) {
  const [expanded, setExpanded] = useState<number | null>(null)

  if (summaries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <Sparkles className="mb-3 h-8 w-8 text-muted-foreground/30" />
        <p className="text-sm text-muted-foreground">在设置中开启 AI 总结</p>
      </div>
    )
  }

  return (
    <div className="space-y-px">
      {summaries.map((summary) => (
        <div key={summary.id} className="-mx-3">
          <button
            className="flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-left transition-colors hover:bg-muted/50"
            onClick={() => setExpanded(expanded === summary.id ? null : summary.id)}
          >
            <div>
              <div className="text-[15px] text-foreground">{summary.title}</div>
              <div className="mt-0.5 text-xs text-muted-foreground">
                {summary.date} · {summary.articleCount} 篇
              </div>
            </div>
            {expanded === summary.id ? (
              <ChevronUp className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            )}
          </button>
          {expanded === summary.id && (
            <div className="px-3 pb-4">
              <div className="prose prose-sm prose-neutral dark:prose-invert max-w-none">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>
                  {summary.summary}
                </ReactMarkdown>
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
