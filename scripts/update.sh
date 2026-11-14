#!/usr/bin/env bash
# 服务器端自动更新（拉模式）：拉最新镜像 → 有变化才滚动重启 → 健康检查 → 失败自动回退旧镜像。
#
# 被谁调用：
#   - 服务器 cron 定时跑（推荐，见 DEPLOY.md「自动更新」）：
#       */10 * * * * /bin/bash /opt/qsl/scripts/update.sh >> /opt/qsl/update.log 2>&1
#   - 也可以人肉跑：bash scripts/update.sh
#
# 幂等：镜像没变化时直接退出，不重启容器，cron 反复跑无副作用。
#
# 前提：部署目录里有 docker-compose.yml（image: ghcr.io/amnssb/hamqsl:latest）。
# 目录可被 DEPLOY_DIR 环境变量覆盖。

set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/qsl}"
cd "$DEPLOY_DIR" || { echo "[x] 部署目录不存在: $DEPLOY_DIR"; exit 1; }

COMPOSE="docker compose"
if ! docker compose version >/dev/null 2>&1; then
  COMPOSE="docker-compose"
fi

# --- 记住当前镜像：回退时就回到它 ---
# 用镜像引用直接查本地 digest，不用 docker compose images ——
# 后者依赖正在运行的容器，容器没跑时会返回空，导致每次都误判有更新而重启。
IMAGE_REF="ghcr.io/amnssb/hamqsl:latest"
OLD_IMAGE="$(docker image inspect "$IMAGE_REF" --format '{{.Id}}' 2>/dev/null || true)"
echo "[1/5] 当前镜像: ${OLD_IMAGE:-（本地没有，将拉取）}"

# --- 拉新镜像 ---
echo "[2/5] 拉取 $IMAGE_REF ..."
if ! $COMPOSE pull; then
  echo "[x] 拉取失败，保持旧版本继续运行。"
  exit 1
fi

NEW_IMAGE="$(docker image inspect "$IMAGE_REF" --format '{{.Id}}' 2>/dev/null || true)"
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
