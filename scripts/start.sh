#!/bin/bash
# QSL 卡片管理系统启动脚本

set -e

echo "=== QSL 卡片管理系统 ==="
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "错误: 请先安装 Docker"
    exit 1
fi

if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
    echo "错误: 请先安装 Docker Compose"
    exit 1
fi

# 复制环境变量
if [ ! -f .env ]; then
    echo "创建 .env 文件..."
    cp .env.example .env 2>/dev/null || true
fi

# 启动服务
echo "启动服务..."
docker compose up -d

echo ""
echo "等待服务启动..."
sleep 10

echo ""
echo "=== 服务已启动 ==="
echo "前端: http://localhost"
echo "API:  http://localhost:8000"
echo "文档: http://localhost:8000/docs"
echo ""
echo "默认账号: admin / admin123"
echo ""
echo "查看日志: docker compose logs -f"
echo "停止服务: docker compose down"
