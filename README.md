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

6. **Cron 任务配置**
   
   **注意**：Vercel 免费套餐不支持 Cron 任务。默认配置中 `vercel.json` 的 `crons` 数组为空，需要手动触发。
   
   如果需要启用自动 Cron（需要 Vercel Pro 套餐），可以：
   - 复制 `vercel.json.example` 的内容到 `vercel.json`
   - 或者手动添加 cron 配置
   
   **手动触发 Cron 任务**：
   
   由于 Vercel 免费套餐不支持 Cron，你可以通过以下方式手动触发：
   
   ```bash
   # 刷新 RSS 订阅
   curl -H "Authorization: Bearer $CRON_SECRET" https://your-project.vercel.app/api/cron/refresh-feeds
   
   # 生成 AI 总结
   curl -H "Authorization: Bearer $CRON_SECRET" https://your-project.vercel.app/api/cron/ai-summary
   
   # 清理过期文章
   curl -H "Authorization: Bearer $CRON_SECRET" https://your-project.vercel.app/api/cron/cleanup
   ```
   
   **使用 GitHub Actions 作为外部 Cron**：
   
   1. 复制 `.github/workflows/cron.yml.example` 到 `.github/workflows/cron.yml`
   2. 在 GitHub 仓库设置中添加 Secrets：
      - `CRON_SECRET`: 你的 Cron 密钥（与 Vercel 环境变量中的 `CRON_SECRET` 相同）
      - `APP_URL`: 你的 Vercel 应用 URL（如 `https://your-project.vercel.app`）
   3. GitHub Actions 会自动按计划执行 Cron 任务
   
   或者使用其他外部 Cron 服务（如 cron-job.org、EasyCron 等）定期调用这些 API。

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
- ⚠️ **注意**：Vercel 免费套餐不支持 Cron 任务，默认配置已禁用自动 Cron
- 💡 **手动触发**：可以通过 curl 或外部 Cron 服务手动触发 Cron 任务（见上方说明）
- 💡 **启用自动 Cron**：如需启用自动 Cron，需要 Vercel Pro 套餐，并参考 `vercel.json.example` 配置

### Docker 部署

Docker 部署支持完整的 Cron 任务功能，适合需要定时任务的场景。

#### 1. 快速开始（开发环境）

使用 Docker Compose 一键启动，包含 PostgreSQL 数据库和 Cron 任务：

```bash
# 克隆项目
git clone <your-repo-url>
cd rssy

# 创建 .env 文件（可选，使用默认值）
cat > .env << EOF
AUTH_SECRET=$(openssl rand -base64 32)
CRON_SECRET=$(openssl rand -base64 32)
AUTH_GITHUB_ID=your-github-client-id
AUTH_GITHUB_SECRET=your-github-client-secret
EOF

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

服务启动后：
- 应用访问：http://localhost:3000
- PostgreSQL：localhost:5432
- 数据库会自动初始化

#### 2. 生产环境部署

##### 方式一：使用外部 PostgreSQL 数据库

如果你已有 PostgreSQL 数据库（如云服务），使用 `docker-compose.prod.yml`：

```bash
# 创建生产环境 .env 文件
cat > .env << EOF
DATABASE_URL=postgresql://user:password@host:port/database
AUTH_SECRET=$(openssl rand -base64 32)
CRON_SECRET=$(openssl rand -base64 32)
AUTH_GITHUB_ID=your-github-client-id
AUTH_GITHUB_SECRET=your-github-client-secret
APP_URL=https://your-domain.com
PORT=3000
EOF

# 启动应用和 Cron
docker-compose -f docker-compose.prod.yml up -d

# 查看日志
docker-compose -f docker-compose.prod.yml logs -f
```

##### 方式二：使用 Docker Compose（包含数据库）

直接使用 `docker-compose.yml`，但需要修改数据库密码：

```bash
# 修改 docker-compose.yml 中的数据库密码
# 然后启动
docker-compose up -d
```

#### 3. 单独构建和运行

```bash
# 构建镜像
docker build -t rssy:latest .

# 运行容器
docker run -d \
  --name rssy-app \
  -p 3000:3000 \
  -e DATABASE_URL=postgresql://user:password@host:port/database \
  -e AUTH_SECRET=your-secret \
  -e CRON_SECRET=your-cron-secret \
  -e AUTH_GITHUB_ID=your-github-id \
  -e AUTH_GITHUB_SECRET=your-github-secret \
  rssy:latest
```

#### 4. Cron 任务说明

Docker 部署包含独立的 Cron 容器，自动执行以下任务：

- **刷新 RSS**：每 30 分钟执行一次 (`*/30 * * * *`)
- **AI 总结**：每小时执行一次 (`0 * * * *`)
- **清理过期文章**：每天凌晨 4 点执行 (`0 4 * * *`)

Cron 容器通过 HTTP 请求调用应用的 API 端点，使用 `CRON_SECRET` 进行认证。

#### 5. 环境变量

| 变量名 | 必需 | 说明 |
|--------|------|------|
| `DATABASE_URL` | 是 | PostgreSQL 数据库连接字符串 |
| `AUTH_SECRET` | 是 | NextAuth.js 的密钥（使用 `openssl rand -base64 32` 生成） |
| `CRON_SECRET` | 是 | Cron 任务认证密钥（使用 `openssl rand -base64 32` 生成） |
| `AUTH_GITHUB_ID` | 是 | GitHub OAuth Client ID |
| `AUTH_GITHUB_SECRET` | 是 | GitHub OAuth Client Secret |
| `APP_URL` | 否 | 应用 URL（Cron 容器使用，默认 `http://app:3000`） |
| `PORT` | 否 | 应用端口（默认 3000） |

#### 6. 数据持久化

使用 Docker Compose 时，PostgreSQL 数据会保存在 `postgres_data` volume 中：

```bash
# 查看 volumes
docker volume ls

# 备份数据
docker run --rm -v rssy_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres-backup.tar.gz /data

# 恢复数据
docker run --rm -v rssy_postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/postgres-backup.tar.gz -C /
```

#### 7. 更新应用

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose build
docker-compose up -d

# 或者使用生产配置
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d
```

#### 8. 故障排查

```bash
# 查看所有容器状态
docker-compose ps

# 查看应用日志
docker-compose logs app

# 查看 Cron 日志
docker-compose logs cron

# 查看数据库日志
docker-compose logs postgres

# 进入应用容器
docker-compose exec app sh

# 手动运行数据库迁移
docker-compose exec app pnpm db:push
```

#### 9. 注意事项

- ✅ Docker 部署支持完整的 Cron 功能
- ✅ 使用 PostgreSQL 数据库（生产环境推荐）
- ✅ 确保 `AUTH_SECRET` 和 `CRON_SECRET` 是强随机密钥
- ✅ 生产环境不要设置 `DEV_MODE=true`
- ⚠️ 首次启动会自动运行 `pnpm db:push` 初始化数据库
- ⚠️ 确保 Cron 容器的 `APP_URL` 能正确访问应用容器
