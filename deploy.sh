#!/usr/bin/env bash
# QSL 卡片管理系统 一键容器部署（自动拉取基础镜像、构建、启动）
# 用法：把整个项目目录传到服务器后，在项目根目录执行  bash deploy.sh
set -e

cd "$(dirname "$0")"

if ! command -v docker >/dev/null 2>&1; then
  echo "[x] 未检测到 Docker，请先安装：curl -fsSL https://get.docker.com | bash"
  exit 1
fi

# docker compose v2 插件优先，兼容旧版 docker-compose
COMPOSE="docker compose"
if ! docker compose version >/dev/null 2>&1; then
  COMPOSE="docker-compose"
fi

mkdir -p data uploads

echo "[1/3] 拉取预构建镜像 ghcr.io/amnssb/hamqsl:latest ..."
if ! $COMPOSE pull; then
  echo "[!] 预构建镜像拉取失败（网络受限或包尚未发布），回退为本地构建（首次约 3-5 分钟）..."
  $COMPOSE build --pull
fi

echo "[2/3] 启动容器..."
$COMPOSE up -d

echo "[3/3] 等待服务就绪..."
for i in $(seq 1 30); do
  if curl -fsS -o /dev/null http://127.0.0.1:3073/ 2>/dev/null; then
    echo ""
    echo "=============================================="
    echo " 部署完成！访问 http://服务器IP:3073"
    echo " 默认账号 admin / admin123（请立即修改）"
    echo " 上线前：设置→站点设置→站点地址 改成正式域名"
    echo " 数据目录: $(pwd)/data  （备份拷它就行）"
    echo "=============================================="
    exit 0
  fi
  sleep 2
done

echo "[x] 30 秒内未就绪，查看日志：docker logs qsl"
exit 1
