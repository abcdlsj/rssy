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

## GitHub OAuth 配置

NextAuth.js 使用 `/api/auth/[...nextauth]` 动态路由自动处理所有认证相关请求，包括 OAuth callback。

**Callback URL 格式**：`/api/auth/callback/{provider}`，其中 `{provider}` 是 provider 的 id（GitHub provider 的 id 是 `github`）。

1. 访问 [GitHub Developer Settings](https://github.com/settings/developers)
2. 点击 "New OAuth App" 创建新应用
3. 填写应用信息：
   - **Application name**: RSSy（或自定义名称）
   - **Homepage URL**: `https://yourdomain.com`（生产环境）或 `http://localhost:3000`（开发环境）
   - **Authorization callback URL**: 
     - 生产环境：`https://yourdomain.com/api/auth/callback/github`
     - 开发环境：`http://localhost:3000/api/auth/callback/github`
4. 创建后，复制 **Client ID** 和 **Client Secret** 到环境变量中

## 部署

支持 Vercel 部署，已配置 Cron 任务自动刷新 RSS 和生成 AI 总结。
