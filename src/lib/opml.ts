interface OPMLFeed {
  title: string
  xmlUrl: string
}

export function parseOPML(xml: string): OPMLFeed[] {
  const feeds: OPMLFeed[] = []
  const outlineRegex = /<outline[^>]*>/gi
  const matches = xml.match(outlineRegex) || []

  for (const match of matches) {
    const xmlUrlMatch = match.match(/xmlUrl=["']([^"']+)["']/i)
    const titleMatch = match.match(/(?:title|text)=["']([^"']+)["']/i)

    if (xmlUrlMatch) {
      feeds.push({
        title: titleMatch?.[1] || xmlUrlMatch[1],
        xmlUrl: xmlUrlMatch[1],
      })
    }
  }

  return feeds
}

export function generateOPML(feeds: { title: string; url: string }[]): string {
  const outlines = feeds
    .map(
      (f) =>
        `    <outline type="rss" text="${escapeXml(f.title)}" title="${escapeXml(f.title)}" xmlUrl="${escapeXml(f.url)}" />`
    )
    .join("\n")

  return `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>RSSy Subscriptions</title>
    <dateCreated>${new Date().toISOString()}</dateCreated>
  </head>
  <body>
${outlines}
  </body>
</opml>`
}

function escapeXml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;")
}
