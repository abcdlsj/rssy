import { NextRequest, NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { prisma } from "@/lib/prisma"
import { parseOPML, generateOPML } from "@/lib/opml"
import Parser from "rss-parser"

const parser = new Parser()

export async function GET() {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const feeds = await prisma.feed.findMany({
    where: { userId: session.user.id },
    select: { title: true, url: true },
  })

  const opml = generateOPML(feeds)

  return new NextResponse(opml, {
    headers: {
      "Content-Type": "application/xml",
      "Content-Disposition": `attachment; filename="rssy-${new Date().toISOString().split("T")[0]}.opml"`,
    },
  })
}

export async function POST(request: NextRequest) {
  const session = await auth()
  if (!session?.user?.id) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const formData = await request.formData()
  const file = formData.get("file") as File | null

  if (!file) {
    return NextResponse.json({ error: "No file" }, { status: 400 })
  }

  const xml = await file.text()
  const feeds = parseOPML(xml)

  let imported = 0
  let skipped = 0

  for (const feed of feeds) {
    const exists = await prisma.feed.findFirst({
      where: { url: feed.xmlUrl, userId: session.user.id },
    })

    if (exists) {
      skipped++
      continue
    }

    try {
      const parsed = await parser.parseURL(feed.xmlUrl)
      await prisma.feed.create({
        data: {
          url: feed.xmlUrl,
          title: parsed.title || feed.title,
          userId: session.user.id,
        },
      })
      imported++
    } catch {
      skipped++
    }
  }

  return NextResponse.json({ imported, skipped, total: feeds.length })
}
