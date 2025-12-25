# 使用 Node.js 20 作为基础镜像
FROM node:20-alpine AS base

# 安装 pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

# 安装依赖
RUN pnpm install --frozen-lockfile

# 复制 Prisma schema
COPY prisma ./prisma

# 生成 Prisma Client（使用 PostgreSQL schema）
RUN pnpm db:generate

# 复制源代码
COPY . .

# 构建应用
RUN pnpm build

# 生产镜像
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# 安装 pnpm（用于运行 db:push）
RUN corepack enable && corepack prepare pnpm@latest --activate

# 复制必要的文件
COPY --from=base /app/public ./public
COPY --from=base /app/.next/standalone ./
COPY --from=base /app/.next/static ./.next/static
COPY --from=base /app/prisma ./prisma
COPY --from=base /app/package.json ./
COPY --from=base /app/pnpm-lock.yaml ./

# 安装依赖（用于运行 prisma 命令）
RUN pnpm install --prod --frozen-lockfile

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]

