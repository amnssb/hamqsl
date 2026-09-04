#!/bin/bash
# QSL 数据库备份脚本

BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/qsl_backup_$TIMESTAMP.sql"

mkdir -p "$BACKUP_DIR"

echo "备份数据库到 $BACKUP_FILE ..."
docker compose exec -T db pg_dump -U qsl qsl_management > "$BACKUP_FILE"

echo "备份完成: $BACKUP_FILE"
echo "文件大小: $(du -h "$BACKUP_FILE" | cut -f1)"
