#!/bin/bash
# QSL 数据库备份脚本
#
# 默认部署是 SQLite 单文件（./data/qsl.db），所以直接拷文件即可。
# 原版这里跑的是 `docker compose exec -T db pg_dump ...`：项目默认没有 db 服务、
# 也没有 pg_dump，执行必失败——改成按实际部署形态工作。
#
# 用法：
#   bash scripts/backup.sh                 # 备份到 ./backups/
#   bash scripts/backup.sh /mnt/nas/qsl    # 备份到指定目录
#
# crontab 每日 3 点：
#   0 3 * * * cd /opt/qsl && bash scripts/backup.sh >> /var/log/qsl-backup.log 2>&1

set -euo pipefail

cd "$(dirname "$0")/.."

BACKUP_DIR="${1:-./backups}"
DB_FILE="${DB_PATH:-./data/qsl.db}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"

mkdir -p "$BACKUP_DIR"

if [ ! -f "$DB_FILE" ]; then
  echo "[x] 找不到数据库文件：$DB_FILE" >&2
  echo "    如果用的是 PostgreSQL，请改用 pg_dump（见 DEPLOY.md 第 10 节）。" >&2
  exit 1
fi

# 用 SQLite 官方在线备份命令最稳：WAL 模式下直接 cp 可能拷到"半个事务"。
# sqlite3 不在时退回 cp，并提示风险。
OUT="$BACKUP_DIR/qsl-$TIMESTAMP.db"

if command -v sqlite3 >/dev/null 2>&1; then
  # .backup 失败时 sqlite3 仍会创建目标文件（0 字节），所以失败要清掉，
  # 否则备份目录里会静静冒出一堆空文件，真出事时才发现没数据。
  if ! sqlite3 "$DB_FILE" ".backup '$OUT'"; then
    rm -f "$OUT"
    echo "[x] 备份失败：$DB_FILE 不是可读的 SQLite 数据库。" >&2
    echo "    如果部署用的是 PostgreSQL，请改用 pg_dump（见 DEPLOY.md 第 10 节）。" >&2
    exit 1
  fi
  echo "[ok] 已备份（在线一致快照）: $OUT"
else
  cp "$DB_FILE" "$OUT"
  echo "[ok] 已备份（直接拷贝）: $OUT"
  echo "[!] 未安装 sqlite3，WAL 模式在写入瞬间可能有极小误差。"
  echo "    想更稳：apt-get install -y sqlite3，或先 docker compose stop 再拷。"
fi

# 空文件说明备份没成功，别再往下报"完成"
if [ ! -s "$OUT" ]; then
  echo "[x] 备份产物为空：$OUT" >&2
  rm -f "$OUT"
  exit 1
fi

# 上传的图片也一起备（卡片版本图丢了列表就是一片空白）
if [ -d "./uploads" ]; then
  tar -czf "$BACKUP_DIR/uploads-$TIMESTAMP.tar.gz" -C . uploads
  echo "[ok] 已备份上传目录: $BACKUP_DIR/uploads-$TIMESTAMP.tar.gz"
fi

echo "文件大小: $(du -h "$OUT" | cut -f1)"

# 只保留最近 30 份，避免把磁盘塞满
KEEP=30
for pattern in "qsl-*.db" "uploads-*.tar.gz"; do
  # shellcheck disable=SC2012
  find "$BACKUP_DIR" -maxdepth 1 -name "$pattern" -type f -printf '%T@ %p\n' 2>/dev/null \
    | sort -rn | tail -n +$((KEEP + 1)) | cut -d' ' -f2- \
    | while read -r old; do rm -f "$old"; echo "[clean] 已删除旧备份: $old"; done
done

echo "完成。"
