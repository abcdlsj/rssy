# RSSy (Next.js)

使用 Next.js + TypeScript + Tailwind CSS 重构的 RSS 阅读器。

## 技术栈

- Next.js 16 (App Router)
- TypeScript
- Tailwind CSS
- SQLite (Prisma ORM)
- NextAuth.js (GitHub OAuth)
- OpenAI API

## 功能

- GitHub OAuth 登录
- RSS 订阅管理
- 文章列表和阅读
- AI 每日总结
- 自动清理过期文章

## 开发

```bash
pnpm install
pnpm prisma db push
pnpm dev
```

## 环境变量

```
DATABASE_URL=file:./prisma/dev.db
AUTH_SECRET=your-secret
AUTH_GITHUB_ID=your-github-client-id
AUTH_GITHUB_SECRET=your-github-client-secret
CRON_SECRET=your-cron-secret
```

## 部署

项目可以部署到 Vercel，已配置 Cron 任务自动刷新 RSS 和生成 AI 总结。
