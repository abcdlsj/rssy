import { generateText } from "ai"
import { createOpenAI } from "@ai-sdk/openai"

const DEFAULT_PROMPT = `请分析并总结今天的RSS文章内容：

1. 整体概述：简要概括主要话题
2. 分类整理：按主题分类
3. 重点摘要：挑选3-5篇最有价值的文章
4. 关键观点：提取关键见解

使用清晰的结构化格式输出。`

interface Article {
  title: string
  link: string
  content?: string | null
  feedTitle?: string
}

export async function generateAISummary(
  articles: Article[],
  apiKey: string,
  endpoint?: string,
  customPrompt?: string
): Promise<{ title: string; summary: string }> {
  if (articles.length === 0) {
    return {
      title: "今日无新文章",
      summary: "今天没有新的RSS文章需要总结。",
    }
  }

  const openai = createOpenAI({
    apiKey,
    baseURL: endpoint || "https://api.openai.com/v1",
  })

  const articlesText = articles
    .map(
      (a, i) =>
        `${i + 1}. ${a.title}\n   来源: ${a.feedTitle || "未知"}\n   摘要: ${(a.content || "").slice(0, 300)}`
    )
    .join("\n\n")

  const prompt = customPrompt || DEFAULT_PROMPT

  try {
    const { text } = await generateText({
      model: openai("gpt-4o-mini"),
      system: "你是RSS文章分析助手，擅长总结和分类文章内容。",
      prompt: `${prompt}\n\n今天的RSS文章（共${articles.length}篇）：\n\n${articlesText}`,
    })

    return {
      title: `${new Date().toLocaleDateString("zh-CN")} RSS 日报`,
      summary: text,
    }
  } catch (error) {
    console.error("AI Summary error:", error)
    return {
      title: `${new Date().toLocaleDateString("zh-CN")} 文章汇总`,
      summary: `今日共有 ${articles.length} 篇文章：\n\n${articles.map((a) => `- ${a.title}`).join("\n")}`,
    }
  }
}
