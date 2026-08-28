# 宝塔部署说明 (Go 项目)

## 产物结构
```
blog-server/                  # 直接 scp/rsync 到 /www/wwwroot/blog-server/
├── blog-server               # Linux 静态二进制 (已 chmod +x)
├── config/
│   └── config.local.yaml     # 生产配置 (优先级最高,已在 .gitignore)
├── start.sh / stop.sh        # 启停脚本 (备用)
└── migrations/               # 如需执行 DB 迁移
```

## 上传到服务器
```bash
# 在本机 (PowerShell / Git Bash)
scp -r blog-server/ root@<服务器IP>:/www/wwwroot/

# 服务器上
cd /www/wwwroot/blog-server
chmod +x blog-server start.sh stop.sh
```

## 宝塔面板 -> 网站 -> 添加 Go 项目
- **项目执行文件**: `/www/wwwroot/blog-server/blog-server`
- **项目名称**: `blog-server`
- **项目端口**: `8080`
- **执行命令**: 留空 (宝塔默认直接执行上面的可执行文件)
- **运行用户**: `www`
- **环境变量**: 无 (程序会自动加载 `config/config.local.yaml`)
- **开机启动**: 勾选

提交后宝塔会自己拉起进程、监控端口 8080。

## 首次部署: 替换 JWT secret
```bash
cd /www/wwwroot/blog-server
openssl rand -hex 32   # 拷输出替换 config/config.local.yaml 里的 jwt.secret
# 然后宝塔面板里点"重启项目"
```

## 健康检查
```bash
curl http://127.0.0.1:8080/healthz   # 若无此路由则返回 404,属正常
tail -f /www/wwwroot/blog-server/logs/run.log   # 如果用 start.sh 启的
```
