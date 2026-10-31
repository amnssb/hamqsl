# 账号体系升级：自定义用户名 + 二步验证 + 通行密钥

> 面向实施者（人或 AI）。读完这份就能动手，不需要先猜设计意图。
> 适用代码：本仓库 HEAD `1e097d7`，Go + Gin + GORM + SQLite，前端 Vue3 + Pinia。

---

## 实施状态：已落地（本文现在描述的是"已实现的东西"）

三期一起做完了，验证脚本 `scripts/smoke-auth.ps1`（19 项，全过）：

```bash
powershell -ExecutionPolicy Bypass -File scripts/smoke-auth.ps1
```

| 交付 | 位置 |
|---|---|
| 真实客户端地址（不再信 XFF） | `internal/middleware/clientip.go` |
| 登录双维度限速（IP + 账号）+ 验证码限速 | `internal/middleware/ratelimit.go` |
| 令牌版本、amr、请求级账号校验 | `internal/middleware/auth.go` |
| 登录三岔口、二步验证、恢复码 | `internal/handler/{auth,totp,ticket}.go` |
| 通行密钥注册/免密登录/第二因子 | `internal/handler/passkey.go` |
| 改用户名、安全总览、强制下线 | `internal/handler/account.go` |
| 一张表清单驱动建表与备份 | `internal/model/tables.go` |
| 登录两阶段 + 通行密钥按钮 | `frontend/src/views/Login.vue`、`stores/auth.js`、`utils/webauthn.js` |
| 账号安全面板（2FA 二维码/恢复码/通行密钥） | `frontend/src/views/Settings.vue` |

上线要配的三件事（都不配也能跑，只是对应功能关闭）：

1. `WEBAUTHN_RP_ID` / `WEBAUTHN_ORIGINS` —— 不配则通行密钥不可用
2. 反代加 `proxy_set_header X-Real-Client-IP $remote_addr;` —— 不加则所有请求共用 127.0.0.1 的 IP 桶
3. `docker-compose.yml` 端口已改为 `127.0.0.1:3073:8000` —— 源站不再对公网裸露

---

## 0. 先定三件事

**① 不重构目录。** 现有的 `internal/{handler,middleware,model}` 是能跑的结构，为了"更符合整洁架构"
去拍平或重排，收益小于风险。新代码顺着现有位置放，但文件名用**业务主体名**，不用技术角色名：

| 新增文件 | 管什么 |
|---|---|
| `backend/internal/handler/account.go` | 账号本身：改名、改密、会话失效、账号安全总览 |
| `backend/internal/handler/totp.go` | 二步验证（TOTP）：生成、确认、校验、恢复码 |
| `backend/internal/handler/passkey.go` | 通行密钥（WebAuthn）：注册、免密登录、列表、删除 |
| `backend/internal/handler/ticket.go` | 登录票据：密码过了但还没过第二因子时，那次"半登录"的凭据 |
| `backend/internal/model/tables.go` | **唯一一张表清单**（见 ②） |
| `frontend/src/utils/webauthn.js` | base64url ↔ ArrayBuffer 这类浏览器 API 的胶水 |

不叫 `webauthn.go`、`mfa.go`、`auth2.go`：这三者说的是技术，接手者猜不到里面是"通行密钥"。

**② 表清单只写一份（这条会顺手修掉一个隐患）。**

现在 `export.go` 里同一批表写了两遍（`24-36` 行导出、`134-138` 行清空、`168-178` 行恢复），
`main.go:49-54` 的 `AutoMigrate` 又写一遍。加一张新表要改三处，漏一处就是"备份里没有它"或
"导入后旧数据残留"——通行密钥表一旦漏进备份，换机恢复后所有 passkey 消失，人会以为是自己记错了。

抽成 `model/tables.go`：

```go
// 表清单：建表、导出、导入都只读这一份。加表只改这里，不许在别处再列一遍。
type Table struct {
	Name    string // 表名，也是备份 JSON 里的键
	Model   any    // GORM 模型，AutoMigrate 和查询都用它
	Backup  bool   // 是否写进 /api/admin/export
	Restore bool   // 导入时是否先清空再写入
}

var Tables = []Table{
	{"users", &User{}, false, false},        // 账号不进备份：口令哈希不外泄，导入也不覆盖，避免锁死自己
	{"passkeys", &Passkey{}, true, false},   // 公钥可进备份（利于审计），但导入不恢复：见 §7
	{"station_profiles", &StationProfile{}, true, true},
	// …其余照搬现有清单
}
```

`main.go` 改成 `for _, t := range model.Tables { db.AutoMigrate(t.Model) }`，
`export.go` 两处循环都换成遍历 `model.Tables`。

> 备份 JSON 的结构不变（仍是 `{exported_at, tables:{表名: [...]}}`），旧备份照样能导入。
> 这是纯内部重构，对外零变化。

**③ 登录默认走"渐进式"：** 用户绑了第二因子才要求第二因子。
不搞"一把梭强制 2FA"——那会让没绑的人当场登不进去。要强制时用 `REQUIRE_MFA=true` 显式打开。

---

## 1. 三种登录路径

```
① 密码 + 二步验证（绑了 TOTP 或 passkey 时的默认路径）
POST /api/auth/login {username,password}
  ├ 失败 → 401（统一响应体 + dummy bcrypt，耗时对齐）
  ├ 成功、未绑任何第二因子 → 200 {access_token, refresh_token}      ← 现状行为
  └ 成功、已绑 → 200 {mfa_required:true, ticket, methods:["totp","passkey"], expires_in:300}
       ├ POST /api/auth/mfa/totp            {ticket, code}      → 200 tokens
       └ POST /api/auth/mfa/passkey/begin   {ticket}            → 200 选项
         POST /api/auth/mfa/passkey/finish  {ticket, credential} → 200 tokens

② 通行密钥免密登录（不输密码，浏览器直接弹 passkey）
POST /api/auth/passkey/login/begin   {}                    → 200 选项（discoverable，免用户名）
POST /api/auth/passkey/login/finish  {credential}          → 200 tokens

③ 恢复码（TOTP 丢了、设备没了）
POST /api/auth/mfa/recovery {ticket, code}                 → 200 tokens（用掉即作废）
```

**ticket 是什么**：密码已验证通过、但第二因子还没过的短期凭据。32 字节随机数，
落库一次性消费，5 分钟过期。它**不是** token，不能访问任何业务接口——
只是"这个人确实知道口令"的收据。所以它单独一张表，不放 JWT。

为什么不塞进 JWT：JWT 一旦签出去就收不回，而半登录凭据必须能立刻作废（用户中途登出、超限锁定）。
一张表换来的可控性，值这点复杂度。

---

## 2. 数据模型

### 2.1 `users` 加字段（`model/models.go:5-13`）

```go
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	HashPassword string     `gorm:"size:128" json:"-"`   // 去掉 not null：允许纯 passkey 账号
	DisplayName  string     `gorm:"size:100" json:"display_name"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	IsAdmin      bool       `gorm:"default:false" json:"is_admin"`

	TokenVersion int        `gorm:"default:0" json:"-"`  // 一加一，所有已发令牌立即失效
	FailedCount  int        `gorm:"default:0" json:"-"`  // 连续失败次数，成功即清零
	LockedUntil  time.Time  `json:"-"`                   // 锁定到期时间
	LastLoginAt  *time.Time `json:"last_login_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

`TokenVersion` 是关键：现在的"改密后旧刷新令牌失效"是靠比较 `user.UpdatedAt` 与
`issuedAt`（`handler/auth.go:131`）实现的，秒级精度 + GORM 自动更新 `UpdatedAt`，
改任何字段都会误伤。换成显式版本号：签发时写进 claims，校验时比对。

### 2.2 新表

```go
// 通行密钥：一个账号可以有多把（手机、笔记本、备用安全密钥）
type Passkey struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	CredentialID []byte    `gorm:"uniqueIndex;not null" json:"-"` // 浏览器给的句柄
	PublicKey    []byte    `json:"-"`                             // 只有公钥，泄露也不致命
	SignCount    uint32    `json:"-"`                             // 防克隆计数器
	Transports   string    `gorm:"size:100" json:"transports"`     // "usb,nfc,internal"
	AAGUID       []byte    `json:"-"`
	BackupAble   bool      `json:"backup_able"`                    // 是否云同步型 passkey
	Name         string    `gorm:"size:60" json:"name"`            // "我的 iPhone"
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// 二步验证：一个账号一条，secret 加密存储
type TotpSecret struct {
	UserID       uint      `gorm:"primaryKey" json:"user_id"`
	SecretEnc    []byte    `gorm:"not null" json:"-"`  // AES-256-GCM 密文，绝不明文
	Enabled      bool      `gorm:"default:false"`      // 确认过第一次验证码才算 true
	LastUsedStep int64     `gorm:"default:0"`          // 已用过的 30 秒窗口，防同码重放
	RecoveryHash string    `gorm:"type:text" json:"-"` // 10 个恢复码的 bcrypt 哈希，JSON 数组
	CreatedAt    time.Time
	ConfirmedAt  *time.Time
}

// 登录票据：密码过了、第二因子没过的那 5 分钟
type LoginTicket struct {
	ID          string    `gorm:"primaryKey;size:64"`      // 随机串
	UserID      uint      `gorm:"index"`                   // 属于谁
	Purpose     string    `gorm:"size:20"`                 // "totp" | "passkey"
	SessionData string    `gorm:"type:text"`               // WebAuthn begin 的中间态 JSON
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
	IP          string    `gorm:"size:50"`                 // 用 middleware.RealClientIP，不用 c.ClientIP
	CreatedAt   time.Time
}
```

**secret 的加密密钥**放哪：复用 `config.go:51` 已有的 `loadOrCreateSecret` 模式，
生成 `data/.mfa_key`（32 字节 hex，0600）。和 `data/.jwt_secret` 同目录、同生命周期——
备份 `data/` 就一起搬走，不会出现"库搬过去了但密钥没搬"。

---

## 3. 接口契约（前端照这个写，不用猜）

| 方法 | 路径 | 请求 | 成功响应 | 失败 |
|---|---|---|---|---|
| POST | `/api/auth/login` | `{username,password}` | 现状 `{access_token,refresh_token,...}` 或 `{mfa_required:true,ticket,methods,expires_in}` | 401 `QSL-401-0001` |
| POST | `/api/auth/mfa/totp` | `{ticket,code}` | tokens | 401 码错 / 410 票据过期 |
| POST | `/api/auth/mfa/recovery` | `{ticket,code}` | tokens | 401 / 410 |
| POST | `/api/auth/mfa/passkey/begin` | `{ticket}` | `{publicKey:{...}}` | 410 |
| POST | `/api/auth/mfa/passkey/finish` | `{ticket,credential}` | tokens | 401 / 410 |
| POST | `/api/auth/passkey/login/begin` | `{}` | `{publicKey:{...}}` | 400 |
| POST | `/api/auth/passkey/login/finish` | `{credential}` | tokens | 401 |
| GET | `/api/auth/security` | — | `{username,totp_enabled,passkeys:[{id,name,last_used_at}],recovery_left}` | 401 |
| POST | `/api/auth/username` | `{username,password}` | `{access_token,refresh_token}` | 400/401/409 |
| POST | `/api/auth/passkey/register/begin` | `{name}` | `{publicKey:{...}}` | 401 |
| POST | `/api/auth/passkey/register/finish` | `{credential}` | `{id,name}` | 400 |
| DELETE | `/api/auth/passkey/:id` | `{password}` | `{ok:true}` | 400/404 |
| POST | `/api/auth/totp/setup` | — | `{secret,otpauth_url}` | 401 |
| POST | `/api/auth/totp/enable` | `{code}` | `{recovery_codes:[...]}` 仅此一次 | 400 |
| POST | `/api/auth/totp/disable` | `{password,code}` | `{ok:true}` | 400 |
| POST | `/api/auth/sessions/revoke` | `{password}` | `{ok:true}` | 400 |

`410`（而非 401）用于"票据过期/已用"：前端能据此提示"请重新登录"而不是"密码错误"。

改用户名、删 passkey、关 TOTP 都要**再输一次密码**：这些操作一旦被 CSRF 或误触,
等于把账号送人。虽然不是 cookie 认证（CSRF 面小），但顺手做掉更稳。

---

## 4. 后端改动清单

### 4.1 `middleware/auth.go` — 令牌里带上"怎么登的"

`GenerateToken` 加两个 claim：

```go
"ver": user.TokenVersion,   // 对应用户口令版本，比对不一致即失效
"amr": []string{"pwd"},     // 认证方式：pwd / totp / passkey，多因子就多元素
```

`AuthRequired` 里加三步（顺序别调）：

```go
// 1) 版本比对：改密、改名、解绑，都靠这一句踢掉旧令牌
var u model.User
if err := db.First(&u, "username = ?", claims["sub"]).Error; err != nil || !u.IsActive {
	401 // 账号没了或被停用，令牌立刻作废
}
if int(claims["ver"].(float64)) != u.TokenVersion { 401 }

// 2) amr 取出，放进上下文，供敏感接口判断
c.Set("amr", claims["amr"])
```

> `AuthRequired` 现在只验签名不查库（`middleware/auth.go:104-140`），
> 所以删账号、停用账号都拦不住已签发的令牌。上面这步顺手补上。
> SQLite 主键点查是微秒级，这点开销换即时吊销，值得。

新增 `RequireMFA()`：`amr` 里没有 `totp`/`passkey` 就 `403 {"code":"QSL-403-0002","message":"该操作需要二次验证"}`。
挂到 `main.go:189-190`（导出/导入）和 `220-226`（SMTP 设置，里面有凭据）。

**访问令牌有效期从 7200 改 900 秒。** 前端 `api/index.js` 已有 401 自动刷新，
用户无感。短令牌 + 版本号，吊销才算真的吊销。

### 4.2 `middleware/ratelimit.go` — 与上一份安全报告合并

`RateLimitMiddleware` 里的 `ip := c.ClientIP()` 换成 `middleware.RealClientIP(c)`（见
`SECURITY-REVIEW-login-ratelimit.md` 的 D2/D3）。登录再加一个账号维度桶：

```go
// 账号桶和 IP 桶任一超限就拒。伪造 XFF 只能换掉 IP 那一半，账号桶照样锁。
if !LoginIPLimiter.Allow(ip) || !LoginUserLimiter.Allow(strings.ToLower(req.Username)) { 429 }
```

TOTP 校验单独限速：6 位数字只有 100 万种，不限速等于送。
`Allow("totp:"+username)`，5 次 / 5 分钟，超限锁账号 15 分钟（写 `user.LockedUntil`）。

### 4.3 `handler/auth.go` — 登录改成三岔口

把现在 `34-63` 行的 `Login` 拆成"验口令"和"发令牌"两段：

```go
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct{ Username, Password string }
	if err := c.ShouldBindJSON(&req); err != nil { response.Fail(c,400,"参数错误"); return }

	ip := middleware.RealClientIP(c)
	// 锁定检查放在最前：被锁的账号连口令都不验，避免用耗时差异探测
	var user model.User
	err := h.db.Where("username = ?", req.Username).First(&user).Error

	// 用户不存在也跑一次等价 bcrypt，抹平快慢路径（现在的 41-45 行是直接的枚举信道）
	if err != nil || !user.IsActive { dummyBcrypt(req.Password); audit(...); response.Unauthorized(c); return }
	if user.HashPassword == "" || !middleware.CheckPassword(user.HashPassword, req.Password) {
		h.recordFail(&user); audit(...); response.Unauthorized(c); return
	}

	h.clearFail(&user)
	// 绑了任一第二因子 → 只发票据；否则直接发令牌
	if h.hasSecondFactor(user.ID) {
		ticket := ticketIssue(h.db, user, "totp", ip)
		response.OK(c, gin.H{"mfa_required": true, "ticket": ticket, "methods": h.methods(user.ID), "expires_in": 300})
		return
	}
	issueTokens(c, user, []string{"pwd"})
}
```

`issueTokens` 是 `51-62` 行那段抽出来的，`amr` 按实际认证方式传。

### 4.4 `handler/totp.go` / `handler/passkey.go` / `handler/ticket.go`

文件头按这个格式写，接手者打开就知道怎么调：

```go
/*
二步验证：生成密钥、确认启用、校验验证码、恢复码。
校验示例：
  ok := totp.Check(userID, "123456")   // 已启用且未超限才返回 true
  codes := totp.Enable(userID, "123456") // 返回 10 个恢复码，仅此一次可见
*/
```

要点：

- **生成**：`totp.Generate({Issuer:"BA4SLT QSL", AccountName: username})`，只返回 `secret` 和 `otpauth_url`，
  `Enabled=false`。**此时还没生效**——必须用户拿验证码回来确认，否则手机没添加上就锁死了。
- **确认**：校验通过才 `Enabled=true`，同时生成 10 个恢复码（`crypto/rand` → base32，8 字符），
  bcrypt 后存库，明文**只在这一次响应里返回**。
- **校验**：`ServerTime` 用 `time.Now()`，`Skew:1`（容忍 ±30 秒）。
  通过后把 `LastUsedStep` 更新为当前 `unix/30`；如果等于上一次的 step，直接拒——防同码重放。
- **passkey 注册**：`ResidentKey: required`、`UserVerification: required`。
  这两项一起才能支持"免用户名直接登录"，否则浏览器弹不出账号选择器。
- **passkey 登录**：`BeginDiscoverableLogin()`，不需要用户名；
  `FinishDiscoverableLogin()` 回调里用 `credentialID` 查 `passkeys` 表定位用户。
- **signCount**：只在 `stored > 0 && new > 0` 时比较，变小说明可能被克隆，记 `PASSKEY_COUNT_ANOMALY` 审计并拒。
  云同步型 passkey 常年是 0，硬比会误杀正常用户。
- **改用户名不影响 passkey**：passkey 靠 `credential_id → user_id` 关联，跟用户名无关。
  但密码管理器里显示的条目名还是旧的，可以在改名后提示一句"通行密钥在密码管理器里的显示名不会变"。

### 4.5 `main.go` — 路由与启动

`49-54` 的 `AutoMigrate` 换成遍历 `model.Tables`。登录相关路由：

```go
// 认证：全部挂登录限速；只有 auth/me 和账号管理挂 AuthRequired
api.POST("/auth/login",   middleware.LoginRateLimitMiddleware(), authH.Login)
api.POST("/auth/mfa/totp",     middleware.LoginRateLimitMiddleware(), authH.MfaTotp)
api.POST("/auth/mfa/recovery", middleware.LoginRateLimitMiddleware(), authH.MfaRecovery)
api.POST("/auth/mfa/passkey/begin",  middleware.LoginRateLimitMiddleware(), pkH.MfaBegin)
api.POST("/auth/mfa/passkey/finish", middleware.LoginRateLimitMiddleware(), pkH.MfaFinish)
api.POST("/auth/passkey/login/begin",  middleware.LoginRateLimitMiddleware(), pkH.LoginBegin)
api.POST("/auth/passkey/login/finish", middleware.LoginRateLimitMiddleware(), pkH.LoginFinish)
// 绑定与管理：必须已登录
auth.POST("/auth/totp/setup", authH.TotpSetup)   // auth 分组已有 AuthRequired
```

启动段（`62-68`）改两处：
- `ADMIN_PASS` 为空时**生成 20 位随机口令并打印一次**，别再用 `admin123`（`config.go:39`）；
- 加逃生开关 `ADMIN_RESET=1`：启动时清空该账号的 TOTP、passkey、失败计数、`TokenVersion++`，
  打印一次性随机口令。**这是唯一的救命绳**，写进 DEPLOY.md。

### 4.6 `config.go` — 新增配置

```go
WebAuthnRPID:    getEnv("WEBAUTHN_RP_ID", ""),       // 例：qsl.ba4slt.cn
WebAuthnOrigins: getEnv("WEBAUTHN_ORIGINS", ""),     // 逗号分隔：https://qsl.ba4slt.cn
RequireMFA:      getEnv("REQUIRE_MFA", "") == "true",
```

启动时如果两个都为空：**只禁用 passkey 功能并打日志**，不要 fatal。
否则老部署升上来会因为少配一个环境变量直接起不来。

---

## 5. 前端改动清单

### 5.1 `src/api/index.js:57` —— 这一行不改会出事

```js
const isAuthCall = url.includes('/auth/login') || url.includes('/auth/refresh')
```

MFA 接口返回 401（验证码错）时，会被当成"令牌过期"→ 触发 refresh → 失败 → `clearSession()`
→ 跳回登录页，用户输错一位就被踢出去。必须扩成：

```js
const isAuthCall = /\/(auth\/(login|refresh|mfa|passkey))/.test(url)
```

### 5.2 `src/stores/auth.js`

`login()` 现在直接取 `res.access_token`（第 12 行）。改成判断分支：

```js
async function login(username, password) {
	const res = await api.post('/auth/login', { username, password })
	if (res?.mfa_required) return { mfaRequired: true, ticket: res.ticket, methods: res.methods }  // 不落 token
	applyTokens(res); await fetchUser(); return { mfaRequired: false }
}

// 抽出来，三个收尾点（密码+2FA、passkey、恢复码）共用
function applyTokens(res) { token.value = res.access_token; localStorage.setItem('token', res.access_token); /* … */ }
```

### 5.3 `src/views/Login.vue`

两阶段**同页**切换，不新增路由：`mfaRequired` 为真时把表单换成验证码输入，
保留"换个账号"的返回键。避免多一个路由就要多处理"刷新页面后 ticket 没了"。

- 底部加 `<PasskeyButton @success="onPasskeyLogin" />`：`navigator.credentials.get()`。
- **非安全上下文要隐藏**（`window.isSecureContext === false`）：明文 HTTP 的地址
  这种明文地址浏览器直接不暴露 `navigator.credentials`，按钮点了只会报 "not supported"。
- 文案里"喵"这类语气词保持现状，不要顺手改成人话，那不是这次的范围。

### 5.4 `src/views/Settings.vue` 新增「账号安全」

四个卡片：改用户名（要密码）、二步验证（二维码 + 10 个恢复码一次性展示 + 关闭）、
通行密钥（列表：名字、上次使用、删除；添加按钮）、所有设备下线（`sessions/revoke`）。

二维码用 `qrcode` 这个前端小库渲染 `otpauth_url`，不要后端出 PNG——
少一个 Go 依赖，也方便以后换 issuer 文案。

### 5.5 依赖

`frontend/package.json`：`qrcode`（二维码）。
`backend/go.mod`：`github.com/go-webauthn/webauthn`、`github.com/pquerna/otp`。
两个都是纯 Go，不影响现在 `CGO_ENABLED=1` 的 SQLite 构建。

---

## 6. 安全细节（这几条不做，功能等于没加）

1. **TOTP secret 绝不进备份、绝不进日志**。`export.go` 的 users 表本来就 `json:"-"` 掉口令哈希
   （见 `models.go:8`），`SecretEnc` 同样 `json:"-"`。
2. **恢复码一次性**：用掉就从未命中里删掉，`recovery_left` 在账号安全页显示剩余数。
3. **改任何安全设置都 `TokenVersion++`**：改密、改名、开关 TOTP、增删 passkey、强制下线。
   一加一，全端立刻重登，这也顺便解决了"改完密码旧手机还登着"。
4. **TOTP 校验限速**（5 次/5 分钟/账号），否则 6 位数可以被在线穷举。
5. **审计要记全**：`MFA_ENABLED`、`MFA_DISABLED`、`MFA_FAILED`、`PASSKEY_ADDED`、`PASSKEY_REMOVED`、
   `USERNAME_CHANGED`、`SESSIONS_REVOKED`、`PASSKEY_COUNT_ANOMALY`。
   IP 一律用 `RealClientIP`——用 `c.ClientIP()` 记下来的审计日志，攻击者想写什么 IP 就写什么 IP。
6. **至少保留一条可登录路径**：最后一个 passkey 不让删（如果没绑 TOTP 且没密码）；
   关 TOTP 时如果没有任何 passkey 且密码为空，拒绝。

---

## 7. 迁移、兼容与逃生

| 场景 | 行为 |
|---|---|
| 老库升级 | `AutoMigrate` 加列加表，默认值保证老账号 `TokenVersion=0`、`IsActive=true`，行为不变 |
| 老令牌 | 无 `ver` claim，按 0 处理，与老账号一致 → **不会被强制重登**（除非你顺便 bump 一次） |
| 没配 `WEBAUTHN_*` | passkey 功能整体关闭，登录页不显示按钮，其余正常 |
| 备份导出 | 不含 users、不含 totp secret；含 passkeys（公钥，用于审计留痕） |
| 备份导入 | **不恢复 users 和 passkeys**。理由：users 不含哈希，恢复了也登不进；passkeys 的 `user_id` 在导入后可能悬空。导入完成后在响应里提示"通行密钥与二步验证需重新绑定" |
| 忘了密码 + 丢了设备 | `ADMIN_RESET=1` 重启，清空第二因子并打印一次性口令。**写进 DEPLOY.md 第 8 节检查清单** |
| 换域名 | **所有 passkey 失效**（RP ID 绑定）。这是 WebAuthn 的硬约束，不是 bug。所以 RP ID 固定为正式域名，别让任何明文 HTTP 入口参与 |

---

## 8. 分三期做，每期都能单独上线

**一期（先堵洞，1 天）**
`RealClientIP` + 登录双维度限速 + dummy bcrypt + `IsActive` 校验 + `ADMIN_PASS` 随机化。
对应的就是安全报告里的 D2/D3/D4/D5。**这一期跟 2FA 无关，但优先级最高**——没有它，2FA 绑了也挡不住爆破。

**二期（账号 + TOTP，2–3 天）**
`TokenVersion`、改名、TOTP 全套、恢复码、审计、`Settings.vue` 账号安全页。
做完就有"密码 + 动态码"，已经能挡住绝大多数爆破。

**三期（passkey，3–4 天）**
注册、免密登录、第二因子登录、列表管理、`ADMIN_RESET` 逃生。
passkey 是体验升级，安全边际不如二期大，放最后。

多账号 CRUD（`IsAdmin`/`IsActive` 真正生效）单独排，别和这三期混——
它牵涉权限模型，混进来会把 2FA 的上线时间拖长。

---

## 9. 验收（写测试时照抄）

- [ ] 老账号密码登录仍成功，行为与升级前一致
- [ ] 绑定 TOTP 后，密码登录返回 `mfa_required`，不给 token
- [ ] 验证码连错 5 次 → 锁定 15 分钟；正确码在锁定期内也进不去
- [ ] 同一个 6 位码用两次 → 第二次拒（重放防护）
- [ ] 恢复码用掉后不能再用，`recovery_left` 从 10 递减
- [ ] passkey 注册后，passwordless 登录成功且能拿到 `/auth/me`
- [ ] 改用户名后，旧 access/refresh 全部 401，新令牌正常
- [ ] 停用账号（`IsActive=false`）后，其所有令牌立刻 401
- [ ] `ADMIN_RESET=1` 启动后，能用一个随机口令登上，且原 passkey/TOTP 全部清空
- [ ] 导出 → 导入 → 老功能全部正常，且提示"需重新绑定 2FA"
- [ ] `X-Forwarded-For` 伪造不影响登录限速（上一份报告的回归项）
- [ ] 非 HTTPS 环境打开登录页，passkey 按钮不出现，且不报错

---

## 10. 给实施者（尤其是 AI）的几条硬规矩

1. **改动过的旧代码要顺手补齐注释**。`auth.go:34-63` 这段本来没有注释，拆的时候补上——
   下一个人读到它时，看到的是"验口令 / 记失败 / 判第二因子 / 发令牌"四步，不是一堆 if。
2. **不要为"不应该发生的情况"写防御**。`ticket` 查不到就返回 410 并结束，
   不要写 `if ticket == nil { if user == nil { ... } }` 这种套娃。
   检查只在边界做一次：`ShouldBindJSON` 之后。
3. **一个数字只写一次**。TOTP 的 `Period=30`、限速的 5 次、ticket 的 300 秒，
   定义成包级常量并注明为什么是这个值，别在四个文件里各写一遍 `30`。
4. **`Ticket` 的 `SessionData` 用同一把密钥加密**。WebAuthn 中间态里有 challenge，
   明文落库等于把一次性挑战变成可预测值。
5. **改接口前先确认没有别的地方在用**。`/api/auth/login` 的响应形状变了（多了 `mfa_required` 分支），
   前端 `stores/auth.js` 和 `api/index.js` 必须同一次提交里改完，否则线上会白屏。
6. **别为了"方便测试"去开放明文 HTTP 直连入口**。
   它没有安全上下文，passkey 在它上面根本不工作；而且按部署加固的要求，这个端口本该只绑回环。
