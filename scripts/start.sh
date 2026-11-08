#!/bin/bash
# QSL 卡片管理系统启动脚本（已废弃，保留仅作指引）
#
# 这个脚本早期版本面向"前端 + 后端 + PostgreSQL 三容器"的部署，
# 现在项目是单容器单进程（后端直接托管前端、默认 SQLite），
# 用 `docker compose up -d` 或 deploy.sh 一条命令就够，不需要它。
#
#   bash deploy.sh          # 拉镜像 → 启动 → 健康检查（推荐）
#   docker compose up -d    # 手动起
#   docker compose logs -f  # 看日志（初始口令在这里）
#
# 直接执行本脚本会打印提示并退出，避免有人照着旧说明去改 .env.example。

echo "本脚本已废弃：项目现在是单容器部署（默认 SQLite，后端托管前端）。"
echo ""
echo "请改用："
echo "  bash deploy.sh          # 一键：拉镜像 → 启动 → 健康检查"
echo "  docker compose up -d    # 手动启动"
echo "  docker compose logs -f  # 查看日志（首次启动的随机口令在这里）"
echo ""
echo "部署细节见 DEPLOY.md。数据在 ./data（SQLite）与 ./uploads。"
exit 1
