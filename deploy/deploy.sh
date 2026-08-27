#!/usr/bin/env bash
# 一键部署/更新 zzdzz-blog
# 用法:
#   1) cp deploy/.env.example deploy/.env && vim deploy/.env
#   2) bash deploy/deploy.sh
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$DEPLOY_DIR")"
cd "$PROJECT_ROOT"

ENV_FILE="$DEPLOY_DIR/.env"
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.yml"

# ──── 1. 校验 env 文件 ────
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ 未找到 $ENV_FILE"
    echo "   先执行: cp $DEPLOY_DIR/.env.example $ENV_FILE"
    echo "   然后填入真实值"
    exit 1
fi

if grep -q "CHANGE_ME" "$ENV_FILE"; then
    echo "❌ $ENV_FILE 还有未替换的 CHANGE_ME 占位符,请先编辑"
    exit 1
fi

echo "▶ 拉取最新代码..."
git pull --rebase --autostash

echo "▶ 拉取/构建镜像..."
docker compose \
    --env-file "$ENV_FILE" \
    -f "$COMPOSE_FILE" \
    pull --ignore-pull-failures || true

docker compose \
    --env-file "$ENV_FILE" \
    -f "$COMPOSE_FILE" \
    build --pull

echo "▶ 启动服务..."
docker compose \
    --env-file "$ENV_FILE" \
    -f "$COMPOSE_FILE" \
    up -d

echo "▶ 等待健康检查..."
sleep 5
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps

echo ""
echo "✅ 部署完成"
echo "   前端访问: http://服务器IP/"
echo "   后端 ping: curl http://localhost/api/v1/ping"
echo "   日志: docker compose -f $COMPOSE_FILE logs -f"