# QSL 卡片管理系统 —— 一体化镜像（后端单进程同时托管前端）
# 构建：docker build -t qsl-management:latest .
# 运行：docker compose up -d   （宿主端口 3073）

# ---------- 前端构建 ----------
FROM node:20-alpine AS fe
WORKDIR /fe
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# ---------- 后端构建（SQLite 驱动需要 CGO） ----------
FROM golang:1.25-alpine AS be
RUN apk add --no-cache gcc musl-dev
WORKDIR /be
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 go build -ldflags '-s -w' -o /qsl-server ./cmd/server

# ---------- 运行镜像 ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 qsl
ENV TZ=Asia/Shanghai GIN_MODE=release
WORKDIR /app
COPY --from=be /qsl-server ./server
COPY --from=fe /fe/dist ./web/dist
RUN mkdir -p data uploads && chown -R qsl:qsl /app
USER qsl
EXPOSE 8000
VOLUME ["/app/data", "/app/uploads"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8000/ >/dev/null 2>&1 || exit 1
CMD ["./server"]
