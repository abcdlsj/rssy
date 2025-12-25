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

### Vercel 部署

项目已配置支持 Vercel 部署，包含自动 Cron 任务。

#### 1. 准备数据库

Vercel 上需要使用 PostgreSQL 数据库。推荐使用：

- **Vercel Postgres**（推荐）：在 Vercel 项目设置中直接添加
- **其他 PostgreSQL 服务**：如 Supabase、Neon、Railway 等

#### 2. 部署步骤

1. **推送代码到 GitHub**
   ```bash
   git push origin main
   ```

2. **在 Vercel 导入项目**
   - 访问 [Vercel Dashboard](https://vercel.com/dashboard)
   - 点击 "Add New" → "Project"
   - 导入你的 GitHub 仓库

3. **配置环境变量**
   
   在 Vercel 项目设置 → Environment Variables 中添加：

   ```bash
   # 数据库（必须）
   DATABASE_URL=postgresql://user:password@host:port/database
   
   # 认证（必须）
   AUTH_SECRET=your-random-secret-string  # 使用 openssl rand -base64 32 生成
   CRON_SECRET=your-cron-secret-string    # 使用 openssl rand -base64 32 生成
   
   # GitHub OAuth（必须，生产环境不使用 DEV_MODE）
   AUTH_GITHUB_ID=your-github-client-id
   AUTH_GITHUB_SECRET=your-github-client-secret
   
   # 可选：开发模式（生产环境不要设置）
   # DEV_MODE=true
   ```

4. **配置 GitHub OAuth**
   
   参考上面的 "GitHub OAuth 配置" 部分，Callback URL 设置为：
   ```
   https://your-project.vercel.app/api/auth/callback/github
   ```

5. **部署并初始化数据库**
   
   - 首次部署后，需要在 Vercel 的部署日志中查看是否有错误
   - 数据库表会在首次运行时自动创建（通过 Prisma）
   - 或者可以手动运行迁移：
     ```bash
     # 在本地连接生产数据库后运行
     pnpm db:push
     ```

6. **配置 Cron 任务**
   
   项目已通过 `vercel.json` 配置了以下 Cron 任务：
   - **刷新 RSS**：每 30 分钟执行一次
   - **AI 总结**：每小时执行一次（根据用户设置的时间）
   - **清理过期文章**：每天凌晨 4 点执行
   
   Vercel 会自动识别 `vercel.json` 中的 cron 配置。

#### 3. 构建配置

- **Build Command**: `pnpm build`（自动使用 PostgreSQL schema）
- **Output Directory**: `.next`
- **Install Command**: `pnpm install`

#### 4. 注意事项

- ✅ Vercel 上默认使用 PostgreSQL（`schema.prisma`）
- ✅ Cron 任务需要 `CRON_SECRET` 环境变量进行认证
- ✅ 生产环境不要设置 `DEV_MODE=true`
- ✅ 确保 `AUTH_SECRET` 是随机生成的强密钥
- ⚠️ 首次部署后需要等待数据库表创建完成
