#!/usr/bin/env bash
# 宝塔测试环境一键部署脚本 (zzdzz-blog)
# 用法: bash deploy/deploy-bt.sh
# 流程: 编译(后端+前端) -> 上传 -> 服务器备份 -> 替换 -> 重启 -> 健康检查
set -euo pipefail

HOST="101.126.22.219"
KEY="keys/101.126.22.219_id_ed25519"
SSH="ssh -i $KEY -o StrictHostKeyChecking=no root@$HOST"
SCP="scp -i $KEY -o StrictHostKeyChecking=no"
REMOTE_DIR="/www/wwwroot/blog-server"
NGINX_DIST="/www/wwwroot/blog-ui/dist"
HEALTH_URL="http://$HOST"
HEALTH_API="http://$HOST/api/v1/articles?page=1&size=1"

cd "$(dirname "$0")/.."

echo "==> [1/5] 编译后端 (linux/amd64)"
( cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/server-linux-amd64 . )
cp server/bin/server-linux-amd64 deploy/release/blog-server/blog-server

echo "==> [2/5] 编译前端"
( cd web && npm run build >/dev/null )
tar czf /tmp/web-dist.tgz -C web/dist .

echo "==> [3/5] 上传"
$SCP deploy/release/blog-server/blog-server root@$HOST:/tmp/blog-server.new
$SCP /tmp/web-dist.tgz root@$HOST:/tmp/web-dist.tgz

echo "==> [4/5] 服务器端备份 + 替换 + 重启"
# 注意: nohup 子进程必须 </dev/null >/dev/null 2>&1, 否则继承 ssh 会话流导致 ssh 挂住
$SSH bash -s <<'REMOTE'
set -e
STAMP=$(date +%Y%m%d-%H%M%S)
DIR=/www/wwwroot/blog-server
cp $DIR/blog-server $DIR/blog-server.bak-$STAMP
cp -r $DIR/web $DIR/web.bak-$STAMP
# 只留最近 3 份备份
ls -1dt $DIR/blog-server.bak-* 2>/dev/null | tail -n +4 | xargs -r rm -f
ls -1dt $DIR/web.bak-* 2>/dev/null | tail -n +4 | xargs -r rm -rf

chmod +x /tmp/blog-server.new && mv -f /tmp/blog-server.new $DIR/blog-server
rm -rf $DIR/web && mkdir -p $DIR/web && tar xzf /tmp/web-dist.tgz -C $DIR/web
rm -rf /www/wwwroot/blog-ui/dist && mkdir -p /www/wwwroot/blog-ui/dist && tar xzf /tmp/web-dist.tgz -C /www/wwwroot/blog-ui/dist
# data/ 存站点自定义图标等持久文件, 部署只保证目录存在, 绝不覆盖内容
mkdir -p $DIR/data/icon
chown -R www:www $DIR /www/wwwroot/blog-ui

PID=$(ps -ef | grep '\./blog-server' | grep -v grep | awk '{print $2}' | head -1)
[ -n "$PID" ] && kill "$PID" && sleep 2
cd $DIR
nohup su -s /bin/sh www -c 'cd /www/wwwroot/blog-server && exec ./blog-server' </dev/null >>/www/wwwroot/blog-server/blog.log 2>&1 &
sleep 3
ps -ef | grep '\./blog-server' | grep -v grep || { echo "!! 后端进程未启动, 查看 blog.log"; exit 1; }
REMOTE

echo "==> [5/5] 健康检查"
for i in 1 2 3 4 5; do
  sleep 2
  HOME_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$HEALTH_URL" || true)
  API_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$HEALTH_API" || true)
  if [ "$HOME_CODE" = "200" ] && [ "$API_CODE" = "200" ]; then
    echo "部署成功: 首页 $HOME_CODE / API $API_CODE"
    echo "回滚方式: ssh root@$HOST 'cp $REMOTE_DIR/blog-server.bak-* $REMOTE_DIR/blog-server' 后重启进程"
    exit 0
  fi
done

echo "!! 健康检查失败 (首页 $HOME_CODE / API $API_CODE), 请登录服务器查看 $REMOTE_DIR/blog.log"
echo "!! 最新备份: $REMOTE_DIR/blog-server.bak-* 与 $REMOTE_DIR/web.bak-*"
exit 1
