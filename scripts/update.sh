#!/usr/bin/env bash
# 服务器端自动更新：拉最新镜像 → 滚动重启 → 健康检查 → 失败自动回退旧镜像。
#
# 被谁调用：
#   - GitHub Actions 构建新镜像成功后 SSH 过来执行（见 .github/workflows/deploy.yml）
#   - 也可以人肉跑：bash scripts/update.sh
#
# 前提：部署目录里有 docker-compose.yml（image: ghcr.io/amnssb/hamqsl:latest）
# 和 .env（本脚本据此找到目录）。目录可被 DEPLOY_DIR 环境变量覆盖。

set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/qsl}"
cd "$DEPLOY_DIR"

COMPOSE="docker compose"
if ! docker compose version >/dev/null 2>&1; then
  COMPOSE="docker-compose"
fi

# --- 记住当前镜像：回退时就回到它 ---
OLD_IMAGE="$($COMPOSE images -q qsl 2>/dev/null || true)"
echo "[1/5] 当前镜像: ${OLD_IMAGE:-（无，首次部署）}"

# --- 拉新镜像 ---
echo "[2/5] 拉取 ghcr.io/amnssb/hamqsl:latest ..."
if ! $COMPOSE pull; then
  echo "[x] 拉取失败，保持旧版本继续运行。"
  exit 1
fi

NEW_IMAGE="$($COMPOSE images -q qsl 2>/dev/null || true)"
if [ -n "$OLD_IMAGE" ] && [ "$NEW_IMAGE" = "$OLD_IMAGE" ]; then
  echo "[2/5] 镜像没有变化，无需更新。"
  exit 0
fi

# --- 滚动替换 ---
echo "[3/5] 重建容器 ..."
$COMPOSE up -d --remove-orphans

# --- 健康检查：90 秒内 / 必须就绪 ---
echo "[4/5] 健康检查 ..."
OK=""
for i in $(seq 1 30); do
  if curl -fsS -o /dev/null http://127.0.0.1:3073/ 2>/dev/null; then
    OK=1
    break
  fi
  sleep 3
done

if [ -z "$OK" ]; then
  echo "[x] 新版本 90 秒未就绪，回退到旧镜像 ..."
  if [ -n "$OLD_IMAGE" ]; then
    docker tag "$OLD_IMAGE" ghcr.io/amnssb/hamqsl:latest
    $COMPOSE up -d --force-recreate
    echo "[!] 已回退。旧容器在跑，请检查 docker logs qsl 后再试。"
  else
    echo "[!] 首次部署失败，无旧镜像可回退。"
  fi
  exit 1
fi

# --- 清理悬空镜像，留最近 2 份防回退 ---
echo "[5/5] 清理悬空镜像 ..."
docker image prune -f >/dev/null

echo "✓ 更新完成：$(date '+%F %T')  镜像 $NEW_IMAGE"
