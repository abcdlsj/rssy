# RSSy

现代化 RSS 阅读器，支持 AI 智能总结。

## 技术栈

- Next.js 16 (App Router)
- TypeScript
- Tailwind CSS
- Prisma ORM (SQLite / PostgreSQL)
- NextAuth.js
- OpenAI API

## 功能

- RSS 订阅管理
- 文章列表（未读/收藏/已读）
- 侧边栏阅读视图
- AI 每日总结
- 自动清理过期文章
- 支持开发模式（无需 OAuth）

## 开发

```bash
pnpm install
pnpm db:push        # 默认 SQLite
pnpm dev            # 默认 SQLite

# 使用 PostgreSQL
DB=pg pnpm db:push
pnpm dev:pg
```

## 环境变量

```bash
DATABASE_URL=file:./prisma/dev.db   # SQLite
# DATABASE_URL=postgresql://...      # PostgreSQL

AUTH_SECRET=your-secret
CRON_SECRET=your-cron-secret

# GitHub OAuth（生产环境）
AUTH_GITHUB_ID=your-github-client-id
AUTH_GITHUB_SECRET=your-github-client-secret

# 开发模式（跳过 OAuth）
DEV_MODE=true
```

## 部署

支持 Vercel 部署，已配置 Cron 任务自动刷新 RSS 和生成 AI 总结。
