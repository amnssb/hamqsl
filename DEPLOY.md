# 部署指南（Linux 服务器 + Nginx）

架构：**Go 单二进制后端 + Vue 静态前端 + SQLite 文件库**。
前端 axios 走相对路径 `/api`，同域反代即可，无跨域问题。

> **容器一键部署（推荐，1Panel 友好）见文末「容器部署」章节**——单容器单进程，
> 后端直接托管前端，宿主端口 **3073**，1Panel 里手动反代指向 `http://127.0.0.1:3073` 即可。

> 以下为传统裸机部署，以 Ubuntu 22.04 / Debian 12 为例，域名以 `qsl.example.com` 代替。

---

## 0. 前置条件（服务器上执行）

```bash
# Go 1.25（后端原生编译，SQLite 驱动需要 CGO，服务器上直接编译最省事）
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc && source ~/.bashrc

# Node 20（仅构建前端用，可构建完卸载）
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Nginx + gcc（CGO 需要）
sudo apt-get install -y nginx gcc
```

## 1. 上传代码

```bash
# 方式 A：git（推荐，后续更新方便）
git clone <你的仓库地址> /opt/qsl-src

# 方式 B：本地打包上传（在 Windows 本机执行）
# tar 打包或直接 scp 整个项目目录（排除 node_modules、backend/data、backend/uploads）
scp -r D:\amnssb\Documents\hamqsl user@服务器IP:/opt/qsl-src
```

## 2. 编译后端

```bash
cd /opt/qsl-src/backend
CGO_ENABLED=1 go build -o /opt/qsl/server ./cmd/server
mkdir -p /opt/qsl/data /opt/qsl/uploads
```

## 3. 构建前端

```bash
cd /opt/qsl-src/frontend
npm ci
npm run build
sudo mkdir -p /var/www/qsl
sudo cp -r dist/* /var/www/qsl/
```

## 4. 迁移数据（二选一）

**方式 A：直接拷库文件（推荐，完整还原）**

在本机（服务停掉的状态下）把 `backend/data/qsl.db` 上传：

```bash
# 本机 PowerShell
scp D:\amnssb\Documents\hamqsl\backend\data\qsl.db user@服务器IP:/opt/qsl/data/qsl.db
```

**方式 B：备份导入**

服务器首次启动后（空库），登录后台 → 设置 → 数据备份 → 导入本机导出的 JSON 备份。
（登录账号与 SMTP 密钥不受导入影响，见备份功能说明）

## 5. systemd 服务（开机自启 + 崩溃自动拉起）

```bash
sudo tee /etc/systemd/system/qsl.service > /dev/null <<'EOF'
[Unit]
Description=QSL Card Management System
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/qsl
ExecStart=/opt/qsl/server
Restart=always
RestartSec=3
# 端口可在配置里改，默认 8000；仅监听本机，由 Nginx 对外
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now qsl
sudo systemctl status qsl   # 确认 active (running)
```

> 注意：`WorkingDirectory` 必须是 `/opt/qsl`，SQLite 库与 uploads 都按相对路径 `./data`、`./uploads` 落盘。

## 6. Nginx 站点

```bash
sudo tee /etc/nginx/sites-available/qsl > /dev/null <<'EOF'
server {
    listen 80;
    server_name qsl.example.com;

    root /var/www/qsl;
    index index.html;

    # 前端 history 路由回退
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 后端 API 反代
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 20m;   # 备份导入 / ADIF 上传
    }

    # 上传的图片
    location /uploads/ {
        proxy_pass http://127.0.0.1:8000;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/qsl /etc/nginx/sites-enabled/qsl
sudo nginx -t && sudo systemctl reload nginx
```

防火墙只放 80/443，**8000 不对外**：

```bash
sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw enable
```

## 7. HTTPS（可选但建议）

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d qsl.example.com
```

## 8. 上线检查清单

1. 浏览器打开 `http://qsl.example.com` —— 首页正常、申请换卡可提交
2. 登录 `/login`（默认 `admin / admin123`）→ **立即修改密码**（设置→账号安全）
3. **设置→站点设置：站点地址改成 `https://qsl.example.com`**——确认收件二维码、邮件里的链接全靠它，不改生成的都是 localhost 链接
4. 设置→SMTP：填服务器上的授权码 → 发送测试邮件（本机测试通过的配置直接重填一遍即可，密码不会随备份导出）
5. 扫一张卡的确认二维码，用手机验证 `/confirm/编号` 能打开
6. 公告栏、回寄地址勾选按需配置

## 9. 日常更新

```bash
# 服务器上
cd /opt/qsl-src && git pull
cd backend  && CGO_ENABLED=1 go build -o /opt/qsl/server ./cmd/server && sudo systemctl restart qsl
cd ../frontend && npm ci && npm run build && sudo cp -r dist/* /var/www/qsl/
```

数据库结构变更会由后端启动时 AutoMigrate 自动完成，无需手动操作。

## 10. 备份策略

- 定期后台导出 JSON 备份（人肉即可），或
- 直接拷库：`cp /opt/qsl/data/qsl.db /备份目录/qsl-$(date +%F).db`（可加 crontab 每日一次，拷前 `systemctl stop qsl` 或接受极小概率写瞬间的误差）

---

## 附：Windows 服务器部署

1. 拷贝 `backend/server.exe` 到 `D:\qsl\`，建好 `D:\qsl\data\`（放 qsl.db）与 `D:\qsl\uploads\`
2. 前端 dist 拷到 `C:\inetpub\qsl`（IIS）或交给 Nginx for Windows
3. 用 [nssm](https://nssm.cc) 注册服务：`nssm install QSL D:\qsl\server.exe`（AppDirectory 设为 `D:\qsl`）
4. 反代配置同上（IIS 用 URL Rewrite + ARR，或直接装 Nginx for Windows）

---

# 容器部署（一键，推荐）

单容器单进程：镜像内先构建前端 dist、再编译后端，运行时**后端直接托管前端页面**（`web/dist` 存在时自动启用，SPA 回退内建），无 Nginx、无 supervisor。宿主端口 **3073**。

**镜像两种来源（默认第一种）：**
- **GHCR 预构建镜像（推荐）**：push 到 `main` 后 GitHub Actions 自动构建发布 `ghcr.io/amnssb/hamqsl:latest`，部署机 `docker compose pull` 秒级获取，**无需装 Go/Node、不占服务器 CPU**
- **本地构建兜底**：网络拉不动 ghcr 时，解开 `docker-compose.yml` 里的 `build:` 注释即可在部署机现场构建（compose 会自动找 Dockerfile）

> 首次发布镜像后记得把包设为公开：GitHub 头像 → **Packages → hamqsl → Package settings → Danger Zone → Change visibility → Public**，否则匿名 `docker pull` 会 401。

## 一键部署

### 方式 A：1Panel 编排（推荐，拉取编排 → 安装）

1. **拉取项目**：服务器上 `git clone https://github.com/amnssb/hamqsl.git /opt/qsl`，或 1Panel → 文件 上传解压
2. **创建编排**：1Panel → 容器 → 编排 → 创建编排
   - 编排内容：粘贴项目根目录 `docker-compose.yml` 的全部内容（或选择从路径导入）
   - **工作目录：必须选项目根目录 `/opt/qsl`**——数据卷 `./data`、`./uploads` 相对它创建
3. **安装/启动**：1Panel 会执行 `docker compose up -d`，自动拉取 `ghcr.io/amnssb/hamqsl:latest`（约 60MB，几十秒）
4. 完成后容器 `qsl` 起来，监听 **3073**；1Panel → 容器 → 日志 可查看

> 编排装好后要更新版本：服务器上 `git pull && docker compose pull && docker compose up -d`（或在 1Panel 编排页面点「重新部署」；数据在 `./data`，不受影响）。

### 方式 B：SSH 命令行（备选）

```bash
cd /opt/qsl && bash deploy.sh    # 自动拉基础镜像 → 构建 → 启动 → 健康检查
```

### JWT_SECRET 是什么

登录成功后系统发给你浏览器的「通行令牌」（JWT）就是用这个密钥签名的；**谁拿到这个密钥，谁就能伪造任意管理员登录态**，不需要密码。

本项目**默认不需要配置它**：`JWT_SECRET` 留空时，后端首次启动自动生成 64 位随机密钥，持久化到 `data/.jwt_secret`（与数据库同目录、随数据卷持久，且已被 .gitignore 排除，不会进 GitHub）。重启、重建容器、迁移数据目录，密钥都跟着 `data` 走。

- 唯一注意：**删了 `data/.jwt_secret` 会导致所有已登录状态失效**，重新登录即可
- 多实例共享时才需要手动指定同一个 `JWT_SECRET`

## 1Panel 反代

1Panel → 网站 → 创建网站 → 反向代理：
- 主域名：你的域名
- 代理地址：`http://127.0.0.1:3073`
- 然后在网站设置里签发 Let's Encrypt 证书即可

## 数据持久化与备份

| 宿主目录 | 容器内 | 内容 |
|---|---|---|
| `./data` | `/app/data` | **qsl.db 数据库**——备份/迁移拷这个 |
| `./uploads` | `/app/uploads` | 上传文件 |

每日备份一行：`cp data/qsl.db backups/qsl-$(date +%F).db`

## 常用运维

```bash
docker logs -f qsl              # 看日志
docker compose restart          # 重启
git pull && docker compose pull && docker compose up -d   # 更新到最新预构建镜像
docker compose down             # 停止（数据在 ./data，不会丢）
```

## 环境变量

- `JWT_SECRET`：留空则首次启动自动生成（推荐，密钥随 `data` 卷持久）；多实例共享数据时才需手动指定同一个值
- 默认 SQLite（`./data/qsl.db`）；要切 PostgreSQL，放开 compose 里 `DB_*` 注释并加一个 postgres 服务即可（代码已支持）

## 从本机迁移数据到容器

1. 停本地服务，把 `backend/data/qsl.db` 上传到服务器 `./data/qsl.db`
2. `docker compose restart` —— 完成；或空库启动后用后台「备份导入」灌 JSON
