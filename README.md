# QSL 卡片管理系统

业余无线电 QSL 卡片管理系统，参考 [bi1kbu/qsl-management](https://github.com/bi1kbu/qsl-management) 项目设计。

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.25 + Gin + GORM |
| 数据库 | **SQLite（默认，单文件 `backend/data/qsl.db`）**，可通过环境变量切换 PostgreSQL |
| 前端 | Vue 3 + Vite + Element Plus |
| 认证 | JWT（Bearer 双令牌：2 小时访问令牌 + 7 天刷新令牌，密钥自动生成） |
| 容器 | Docker 单容器一键部署（宿主端口 3073，见下方"容器部署"） |

## 功能模块

### 核心业务
- **通联日志管理** - QSO/SWL/EYEBALL 记录增删改查
- **卡片管理** - 创建→制卡→发信→收卡 完整流程状态机（含重发）；支持**通联换卡**：从通联日志选择未建卡记录直接建卡，管理主动寄出的卡片
- **线上换卡** - 公开申请（QSO/EYEBALL/SWL 三种场景，各有必填证据字段；EYEBALL 细分线下补换/网络EYE）→ 后台**分家管理**（QSL/SWL/EYEBALL 独立标签页）→ 审核（记录审核人；拒绝含原因并自动邮件通知对方）→ 创建卡片
- **SWL 反寄流程** - SWL 由对方先寄：申请时填写收件地址（用于接收回寄卡）→ 审批通过自动邮件提醒（附进度链接）→ 管理员从"我的地址"多选地址发送到对方邮箱 → 对方在进度页登记邮寄方式（挂号信填单号/平信）→ 卡片寄达后管理员**登记收卡（生成收卡记录）**并邮件通知对方；回寄卡片**单独按需创建**（发卡记录）并寄出，对方寄出后"发送地址"按钮自动隐藏
- **邮件通知** - SWL 审批通过、回寄地址、收卡确认、回寄卡建单等通知均为 HTML 排版邮件（含进度按钮与页脚）；**发卡时自动邮件提醒收卡人**（单号+进度链接，卡片记录"邮件"菜单可手动补发建卡/发信/对方收卡/快递更新通知）；SMTP 测试邮件支持自发自收
- **进度页地址块** - SWL 回寄地址仅在对方登记寄出前显示，寄出后自动隐藏
- **申请表单** - 场景证据时间字段均标注 UTC+8 且精确到分钟（通联/收听/见面）；SWL 表单内置完整流程说明（地址仅用于接收回寄卡片）
- **线下换卡** - 活动管理、批量线下换卡卡片
- **收卡记录** - 收卡确认、自动匹配发卡记录

### 公开门户（无需登录）
- **换卡申请** `/apply` - 7 步表单：场景选择、卡片版本多选（图片网格）、卡片局/直寄收件地址、换卡理由
- **申请进度** `/status/:code` - 按场景深度适配的时间线，手动刷新，签收后隐藏物流信息：
  - QSO：提交申请 → 审核 → 制卡 → 邮寄 → 签收
  - SWL：提交申请 → 审核 → **您寄出卡**（登记单号）→ **我方收卡** → 制卡回寄 → 签收；可查看回寄地址
  - EYEBALL：提交申请 → 审核 → 制卡 → 交付（见面交付或邮寄）→ 完成
- **快递追踪** `/track/:code` - 按单号查询；跳转快递100 官网按单号自动识别全部承运商（无需选择承运商）
- **确认收件** `/confirm/:code` - 呼号校验，自动生成收卡记录
- **卡片查询** - 按卡片编号查询当前状态

### 配置管理
- 卡片版本管理（支持图片上传与缩略图）
- **我的地址** - 维护多条自己的回寄地址，可多选发送到申请人邮箱
- 设置页：站点名称/站点地址/通知邮箱（新申请实时邮件提醒）、SMTP 邮件配置（真实发送的 HTML 测试邮件，密码留空不覆盖）、**数据备份导出**（全库 JSON，敏感配置剔除）、账号安全（修改密码）
- 时间显示：所有入库时间统一 **UTC+8**，页面展示精确到分钟

> 已移除的界面：卡片局管理、本台设备、本台通信地址、快递100 API 配置（相关后端接口保留以兼容数据）。

### 插件系统

功能与主题皆以插件形式扩展，后台「插件」页一键启停，**即时生效无需重启**：

- **主题插件** - 磨砂漩涡 🍥（磨砂玻璃拟态 × 蓝粉白渐变 × 漩涡切片虚化装饰）、暗色主题、纸墨主题（仿老式 QSL 卡暖黄纸面），后台与公开门户同步换肤；启停状态持久化，升级默认全部禁用（零行为变化）
- **功能插件** - 每日统计（`stats_daily`）：按东八区逐天聚合建卡/发卡/寄出/签收计数，插件页内查看最近 14 天趋势柱状图
- **扩展机制** - 后端 `internal/plugin` 注册表：功能插件向 `/api/ext/<name>/...` 注入路由（自带认证与启停守卫，禁用时 404）；启停持久化在 `system_settings`；主题插件皮肤本体定义在前端 `src/plugins/registry.js`（CSS 变量 + 附加样式表）
- 新增插件三步：后端建包实现 `plugin.Plugin` 接口并注册 → 主题在前端 registry 补条目 → 按需在插件页加专属动作

### 安全特性

- **双令牌认证** - 访问令牌 2 小时 + 刷新令牌 7 天（`POST /api/auth/refresh` 自动续期）；修改密码后旧刷新令牌立即失效；刷新令牌不可充当访问令牌；签名算法固定校验防算法混淆
- **速率限制** - 登录/刷新 5 次/分钟（防暴力破解）、公开接口 60 次/分钟、后台 API 600 次/分钟，超限返回 429
- **CORS 白名单** - 仅放行配置的前端来源，替代通配 `*`
- **安全响应头** - CSP（脚本仅同源、禁止内嵌框架）、nosniff、Referrer-Policy、HSTS（HTTPS 部署后自动生效）
- **审计日志** - 登录成功/失败、修改密码自动记录（含来源 IP），随数据备份导出
- **输入校验** - 全接口统一 400/422 负例覆盖（E2E 验证）；上传文件内容嗅探防扩展名伪装

### 移动端适配

- **后台抽屉导航** - ≤860px 侧栏变为抽屉（汉堡按钮唤出、遮罩点击/路由跳转自动收起），内容区占满屏宽
- **弹窗防溢出** - 全部对话框/确认框窄屏自动收敛到 92vw/90vw，无需逐个改宽度
- **表单纵排** - ≤640px 表单标签自动改为上方排列（筛选等 inline 表单保持横排可换行）
- **列表与分页** - 表格窄屏横向滚动 + 紧凑字号；分页隐藏跳页/每页条数选择；卡片头操作按钮自动换行
- **公开门户** - 申请表单/进度页/登录页/门户首页均已适配单列布局（640px 断点）

### 图片上传
- `POST /api/upload/image`（需登录，multipart）
- 限制 5MB，仅 jpg/jpeg/png/gif/webp，含文件内容嗅探（防扩展名伪装）
- 保存至 `./uploads/`，随机文件名，通过 `/uploads/*` 静态访问

## 快速开始（开发模式）

**后端**：

```bash
cd backend
go build -o server ./cmd/server   # Windows: go build -o server.exe ./cmd/server
./server
# 监听 :8000
```

**前端**：

```bash
cd frontend
npm install
npm run dev
# 监听 :3000，/api 与 /uploads 代理到 :8000
```

**默认账号**：admin / admin123（上线后请立即修改）

访问入口：公开门户 `http://localhost:3000/`，后台 `http://localhost:3000/admin/dashboard`。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| DB_DRIVER | sqlite | sqlite 或 postgres |
| DB_PATH | ./data/qsl.db | SQLite 文件路径（自动创建目录） |
| DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME | localhost/5432/qsl/qsl_secret_2024/qsl_management | 仅 postgres 模式使用 |
| JWT_SECRET | 自动生成 | 留空时首次启动随机生成并持久化到数据目录（`data/.jwt_secret`） |
| SERVER_PORT | 8000 | 后端监听端口 |
| ADMIN_USER / ADMIN_PASS | admin / admin123 | 首次启动自动创建管理员 |

## API 文档

统一响应格式 `{ code, message, data }`；前端 axios 拦截器已解包，组件直接使用 `data`。

### 公开接口（无需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/public/station-cards | 启用中的卡片版本列表 |
| GET | /api/public/station-mail-info | 本台回寄地址（未配置返回 `{}`，不返回 404） |
| GET | /api/public/bureaus | 卡片局列表 |
| POST | /api/public/exchange-online | 提交换卡申请（按场景校验必填字段，同呼号待审核时返回 409） |
| GET | /api/public/exchange-status/:request_code | 申请实时进度；制卡后附带 card_code、flow_status、tracking_* 与 tracking_details |
| POST | /api/public/confirm-receipt | 确认收件（生成收卡记录） |
| POST | /api/public/exchange-return-mail | SWL 反寄登记：对方寄出后提交邮寄方式与单号 |
| GET | /api/public/cards/:card_code | 按编号查卡片状态 |
| GET | /api/public/tracking?tracking_number=NUM | 快递追踪（未找到返回 404；配置快递100 后自动拉取轨迹） |
| GET | /api/public/tracking/:tracking_number | 同上（路径参数形式） |
| GET | /api/public/site-info | 站点名称（公开门户标题；未配置返回默认值） |
| GET | /api/public/plugins | 已启用的主题插件列表（公开页据此应用皮肤） |

> 注：旧版 `/api/public/qso-query` 与 `/api/public/card-query` 已在 v1.1 中删除。

### 认证接口

| 分组 | 接口 |
|------|------|
| 认证 | POST /api/auth/login，GET /api/auth/me，POST /api/auth/refresh（刷新令牌续期），POST /api/auth/change-password |
| 插件 | GET /api/plugins（列表），POST /api/plugins/:name/enable、:name/disable（即时生效）；功能插件路由挂 /api/ext/<name>/... |
| 总览 | GET /api/dashboard/summary |
| 通联 | GET/POST /api/qso-records，GET/PUT/DELETE /api/qso-records/:id，POST /api/qso-records/import（ADIF 导入） |
| 卡片 | GET/POST /api/card-records，GET/PUT/DELETE /api/card-records/:id |
| 卡片流程 | POST /api/card-records/:id/issue, :id/sent, :id/received, :id/resend，POST /api/card-records/from-qso（通联换卡） |
| 快递 | GET/POST /api/card-records/:id/tracking，POST /api/card-records/batch-tracking-update |
| 邮件 | POST /api/card-records/:id/send-mail（按 scene 发送建卡/发信/收卡/快递更新通知） |
| 上传 | POST /api/upload/image（multipart，5MB 限制） |
| 线上换卡 | GET/POST /api/exchange/online/requests，POST :id/approve, :id/reject, :id/create-card, :id/send-address（多选地址发对方邮箱）, :id/receive-return（SWL 收卡登记 → 收卡记录 + 对方邮件通知） |
| 线下换卡 | GET/POST /api/exchange/offline/activities，PUT :id |
| 收卡 | GET/POST /api/receive-records |
| 地址 | GET/POST/PUT/DELETE /api/address/book，/api/address/bureaus |
| 本台 | GET/POST/PUT /api/station/profile，GET/POST/PUT /api/station/cards，GET/POST /api/station/equipments |
| 设置 | GET/POST /api/settings/smtp，POST /api/settings/smtp/test，GET/POST /api/settings/tracking，GET/POST /api/settings/site |

## E2E 验证

`scripts/e2e_verify.ps1`（需先启动后端与前端），**116 项检查全部通过**，覆盖：
登录（含双令牌签发）、三场景申请（含负例与 EYEBALL 细分类型）、场景筛选、审核+审核人、拒绝（含原因与防重复）、随机编号建卡、卡片流程、公开查询回归、确认收件、SWL/EYEBALL 映射、换卡进度接口、图片上传（含静态访问与负例）、已删接口 404、追踪未找到 404、修改密码负例、站点设置（公告栏/回寄地址，测试后恢复原值）、QSO 序列单调性、ADIF 导入与去重、SWL 反寄全流程、发送我的地址、通联换卡、**回寄闭环（按卡开启 → 对方登记回寄 → 后台确认收到 → 自动收卡记录）**、测试邮件守卫、**数据备份导出与导入回灌**、**插件系统（列表/未认证 401/默认全禁用/主题启停即时生效/功能路由守卫 404/未知插件 404）**。

```powershell
& ./scripts/e2e_verify.ps1
```

> 注意：E2E 每次运行会写入测试数据（E2EQSO/E2ESWL/E2EEYE 等呼号）。

## 容器部署（推荐）

单容器单进程：镜像内构建前端 dist，后端直接托管页面（无需 Nginx），宿主端口 **3073**，数据持久化在 `./data` 与 `./uploads`。

镜像默认使用 GitHub Actions 自动构建的预构建镜像 `ghcr.io/amnssb/hamqsl:latest`（push 到 main 即发布，部署机免编译）；拉不动时可解开 `docker-compose.yml` 中 `build:` 注释改为本地构建。

```bash
git clone https://github.com/amnssb/hamqsl.git /opt/qsl
cd /opt/qsl
bash deploy.sh          # 拉预构建镜像 → 启动 → 健康检查（拉取失败自动回退本地构建）
```

或手动：

```bash
docker compose pull
docker compose up -d
```

1Panel / 宝塔用户：创建编排粘贴 `docker-compose.yml`（工作目录选项目根目录），安装后把域名反代到 `http://127.0.0.1:3073` 即可。完整说明见 [DEPLOY.md](DEPLOY.md)。

## 数据备份

SQLite（默认）：后台「设置 → 数据备份」一键导出/导入 JSON；或直接复制 `data/qsl.db`。

PostgreSQL（可选模式）：`pg_dump` / `psql` 常规流程。

## 已知限制

- 快递100 自动轨迹为可选配置，未配置 key 时交互以跳转快递100 官网为主
- 确认收件二维码物流依赖承运商官网/快递100，本系统不内置轨迹爬取

## 许可证

GPL-3.0
