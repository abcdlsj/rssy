"use client"

import { useRef, useState } from "react"
import { useRouter } from "next/navigation"
import { Download, Upload, Loader2 } from "lucide-react"

export function OPMLButtons() {
  const [importing, setImporting] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const router = useRouter()

  const handleExport = () => {
    window.location.href = "/api/opml"
  }

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setImporting(true)
    try {
      const formData = new FormData()
      formData.append("file", file)

      const res = await fetch("/api/opml", {
        method: "POST",
        body: formData,
      })

      const data = await res.json()
      alert(`导入完成：${data.imported} 个成功，${data.skipped} 个跳过`)
      router.refresh()
    } catch {
      alert("导入失败")
    } finally {
      setImporting(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
    }
  }

  return (
    <div className="flex items-center gap-2">
      <input
        ref={fileInputRef}
        type="file"
        accept=".opml,.xml"
        onChange={handleImport}
        className="hidden"
      />
      <button
        onClick={() => fileInputRef.current?.click()}
        disabled={importing}
        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
      >
        {importing ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : (
          <Upload className="h-3.5 w-3.5" />
        )}
        导入
      </button>
      <button
        onClick={handleExport}
        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
      >
        <Download className="h-3.5 w-3.5" />
        导出
      </button>
    </div>
  )
}
