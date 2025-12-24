"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Loader2, Check } from "lucide-react"

interface Preference {
  cleanupExpiredDays: number
  enableAutoCleanup: boolean
  enableAISummary: boolean
  aiSummaryTime: string
  aiSummaryPrompt: string | null
  openAIAPIKey: string
  openAIEndpoint: string | null
}

function Section({
  title,
  children
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="py-6 first:pt-0">
      <h2 className="mb-5 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {title}
      </h2>
      <div className="space-y-5">
        {children}
      </div>
    </section>
  )
}

function Row({
  label,
  description,
  children,
}: {
  label: string
  description?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-8">
      <div className="min-w-0">
        <div className="text-sm text-foreground">{label}</div>
        {description && (
          <div className="mt-0.5 text-xs text-muted-foreground">{description}</div>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  )
}

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-sm text-foreground">{label}</label>
      {children}
    </div>
  )
}

export function SettingsForm({ preference }: { preference: Preference }) {
  const [form, setForm] = useState(preference)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try {
      await fetch("/api/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="max-w-md divide-y divide-border">
      <Section title="清理">
        <Row label="自动清理已读文章">
          <Switch
            checked={form.enableAutoCleanup}
            onCheckedChange={(checked) => setForm({ ...form, enableAutoCleanup: checked })}
          />
        </Row>

        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">保留</span>
          <Input
            type="number"
            value={form.cleanupExpiredDays}
            onChange={(e) =>
              setForm({ ...form, cleanupExpiredDays: parseInt(e.target.value) || 30 })
            }
            className="w-16 text-center"
            min={1}
          />
          <span className="text-muted-foreground">天</span>
        </div>
      </Section>

      <Section title="AI 总结">
        <Row label="每日生成">
          <Switch
            checked={form.enableAISummary}
            onCheckedChange={(checked) => setForm({ ...form, enableAISummary: checked })}
          />
        </Row>

        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">时间</span>
          <Input
            type="time"
            value={form.aiSummaryTime}
            onChange={(e) => setForm({ ...form, aiSummaryTime: e.target.value })}
            className="w-24"
          />
        </div>

        <Field label="API Key">
          <Input
            type="password"
            value={form.openAIAPIKey}
            onChange={(e) => setForm({ ...form, openAIAPIKey: e.target.value })}
            placeholder="sk-..."
          />
        </Field>

        <Field label="Endpoint">
          <Input
            type="text"
            value={form.openAIEndpoint || ""}
            onChange={(e) => setForm({ ...form, openAIEndpoint: e.target.value })}
            placeholder="https://api.openai.com/v1"
          />
        </Field>

        <Field label="提示词">
          <Textarea
            value={form.aiSummaryPrompt || ""}
            onChange={(e) => setForm({ ...form, aiSummaryPrompt: e.target.value })}
            placeholder="自定义 AI 总结提示词..."
          />
        </Field>
      </Section>

      <div className="pt-6">
        <Button onClick={handleSave} disabled={saving}>
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : saved ? (
            <Check className="h-4 w-4" />
          ) : null}
          {saved ? "已保存" : "保存"}
        </Button>
      </div>
    </div>
  )
}
