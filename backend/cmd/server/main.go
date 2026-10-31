package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"qsl-management/internal/config"
	"qsl-management/internal/handler"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/internal/plugin"
	pluginstatsdaily "qsl-management/internal/plugins/stats_daily"
	pluginthemeba "qsl-management/internal/plugins/theme_ba"
	pluginthemedark "qsl-management/internal/plugins/theme_dark"
	pluginthemeglass "qsl-management/internal/plugins/theme_glass"
	pluginthemepaper "qsl-management/internal/plugins/theme_paper"
	pluginthemerhodes "qsl-management/internal/plugins/theme_rhodes"
	"qsl-management/pkg/response"
)

func main() {
	cfg := config.Load()
	middleware.SetSecret(cfg.JWTSecret)

	var db *gorm.DB
	var err error

	if cfg.DBDriver == "postgres" {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		dir := filepath.Dir(cfg.DBPath)
		if dir != "." && dir != "" {
			os.MkdirAll(dir, 0755)
		}
		// SQLite 调优：
		//   journal_mode(WAL)     读写不再互相阻塞，前端并发拉几个接口时明显顺
		//   busy_timeout(5000)    拿不到写锁时等 5s 而不是立刻 database is locked
		//   synchronous(NORMAL)   WAL 下安全，且比 FULL 少一次 fsync
		//   cache_size(-20000)    约 20MB page cache，减少小库的重复读
		// 参数用 _pragma 形式，SQLite 驱动会逐条执行。
		dsn := cfg.DBPath +
			"?_pragma=journal_mode(WAL)" +
			"&_pragma=busy_timeout(5000)" +
			"&_pragma=synchronous(NORMAL)" +
			"&_pragma=cache_size(-20000)" +
			"&_pragma=foreign_keys(1)"
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 连接池：SQLite 单写者模型，读多写少；PG 部署时也能吃到合理上限。
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		if cfg.DBDriver == "postgres" {
			sqlDB.SetMaxOpenConns(25)
			sqlDB.SetMaxIdleConns(10)
		} else {
			// WAL 允许多读，写入串行；这里给读留余量，写由 busy_timeout 排队
			sqlDB.SetMaxOpenConns(8)
			sqlDB.SetMaxIdleConns(4)
		}
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	// 建表只读一份清单（model.Tables），加新表不用改这里
	for _, t := range model.Tables {
		if err := db.AutoMigrate(t.Model); err != nil {
			log.Fatalf("建表失败 %s: %v", t.Key, err)
		}
	}

	// 历史数据归一化：老版本场景类型 ONLINE_EYEBALL -> EYEBALL
	//
	// 以前每次启动都无条件跑这 4 条全表 UPDATE，库变大后每次重启都要写一遍 WAL。
	// 现在先 LIMIT 1 探一下，没有待迁移的行就直接跳过；迁移完成后启动是只读的。
	normalizeLegacySceneTypes(db)

	bootstrapAdmin(cfg, db)
	escapeHatch(cfg, db)

	// 插件系统：注册 → 初始化已启用插件 → 之后挂载管理端点与功能路由。
	pm := plugin.NewManager(db)
	pm.Register(pluginthemedark.New())
	pm.Register(pluginthemepaper.New())
	pm.Register(pluginthemeglass.New())
	pm.Register(pluginthemeba.New())
	pm.Register(pluginthemerhodes.New())
	pm.Register(pluginstatsdaily.New(db))
	pm.InitEnabled()

	r := gin.Default()
	// 关掉 gin 内置的 XFF 解析：客户端塞什么头都不该影响"这个请求是谁发的"。
	// 真实地址统一走 middleware.RealClientIP。
	r.ForwardedByClientIP = false
	if err := r.SetTrustedProxies([]string{"127.0.0.1/32", "::1/128"}); err != nil {
		log.Printf("[warn] 设置可信代理失败: %v", err)
	}
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())

	// 图片上传目录与静态访问
	os.MkdirAll("uploads", 0755)
	r.MaxMultipartMemory = 8 << 20
	r.Static("/uploads", "./uploads")

	// 生产模式托管前端：存在 ./web/dist 时由后端直接提供页面（单容器/单二进制部署无需 Nginx）
	webDir := "./web/dist"
	serveWeb := false
	if _, err := os.Stat(webDir); err == nil {
		serveWeb = true
		r.NoRoute(func(c *gin.Context) {
			// 命中的真实文件直接返回；/api 未注册路由保持 JSON 404；其余回退 index.html（history 路由）
			reqPath := c.Request.URL.Path
			p := filepath.Join(webDir, filepath.Clean("/"+reqPath))
			if st, serr := os.Stat(p); serr == nil && !st.IsDir() {
				// /assets/ 下的文件名带内容 hash → 内容不可变，可以长缓存；
				// 其余静态文件（favicon 等）给一天。
				if strings.HasPrefix(reqPath, "/assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					c.Header("Cache-Control", "public, max-age=86400")
				}
				c.File(p)
				return
			}
			if len(reqPath) >= 5 && reqPath[:5] == "/api/" {
				c.JSON(404, gin.H{"code": "QSL-404", "message": "接口不存在"})
				return
			}
			// index.html 绝不能被缓存，否则发版后用户一直拿到旧入口
			c.Header("Cache-Control", "no-cache")
			c.File(filepath.Join(webDir, "index.html"))
		})
		log.Printf("前端静态目录: %s（单进程托管）", webDir)
	}

	r.GET("/", func(c *gin.Context) {
		if serveWeb {
			// 托管前端时根路径返回门户页面，而不是 API 说明
			c.Header("Cache-Control", "no-cache")
			c.File(filepath.Join(webDir, "index.html"))
			return
		}
		c.JSON(200, gin.H{"message": "QSL 卡片管理系统 API", "version": "1.0.0", "db": cfg.DBDriver})
	})

	// 统一限流：/api 组已经带了 APIRateLimitMiddleware，
	// 子组不要再挂一次——否则公开接口每个请求会消耗两次配额。
	api := r.Group("/api", middleware.APIRateLimitMiddleware())

	// 公开接口（无需登录；限流由上面的 /api 组统一承担）
	pub := api.Group("/public")
	pubH := handler.NewPublicHandler(db)
	pub.GET("/site-info", pubH.SiteInfo)
	pub.GET("/stats", pubH.PublicStats)
	pub.GET("/station-cards", pubH.StationCards)
	pub.GET("/station-mail-info", pubH.StationMailInfo)
	pub.GET("/exchange-status/:request_code", pubH.ExchangeStatus)
	pub.GET("/bureaus", pubH.Bureaus)
	pub.POST("/exchange-online", pubH.SubmitExchange)        // 线上换卡申请
	pub.POST("/confirm-receipt", pubH.ConfirmReceipt)        // 确认收件
	pub.POST("/return-mail", pubH.ReturnMail)                // 回寄登记：对方确认收卡后登记回寄方式与单号
	pub.POST("/exchange-return-mail", pubH.SubmitReturnMail) // SWL 反寄登记
	pub.GET("/cards/:card_code", pubH.GetCardByCode)         // 按编号查卡片（公开）
	pub.GET("/tracking", pubH.QueryTracking)                 // 快递追踪
	pub.GET("/tracking/:tracking_number", pubH.QueryTracking)
	pub.GET("/plugins", pm.HandlePublicThemes) // 已启用主题插件（公开页应用皮肤）
	// 主题配色变量：index.html 用 <link> 预置，首帧即命中正确配色（消除 FOUC）
	pub.GET("/theme.css", pm.HandleThemeCSS)

	// 认证：登录链路全部走严格限速
	authH := handler.NewAuthHandler(db)
	totpH := handler.NewTotpHandler(db)
	accH := handler.NewAccountHandler(db)
	pkH := handler.NewPasskeyHandler(db, newWebAuthn(cfg))

	api.POST("/auth/login", middleware.LoginRateLimitMiddleware(), authH.Login)
	api.POST("/auth/refresh", middleware.LoginRateLimitMiddleware(), authH.Refresh)
	api.POST("/auth/mfa/totp", middleware.LoginRateLimitMiddleware(), authH.MfaTotp)
	api.POST("/auth/mfa/recovery", middleware.LoginRateLimitMiddleware(), authH.MfaRecovery)
	api.POST("/auth/mfa/passkey/begin", middleware.LoginRateLimitMiddleware(), pkH.MfaBegin)
	api.POST("/auth/mfa/passkey/finish", middleware.LoginRateLimitMiddleware(), pkH.MfaFinish)
	api.POST("/auth/passkey/login/begin", middleware.LoginRateLimitMiddleware(), pkH.LoginBegin)
	api.POST("/auth/passkey/login/finish", middleware.LoginRateLimitMiddleware(), pkH.LoginFinish)
	// 登录页要据此决定显不显示"用通行密钥登录"，所以是公开的
	api.GET("/auth/capabilities", func(c *gin.Context) {
		response.OK(c, gin.H{"passkey": pkH.Enabled()})
	})

	api.GET("/auth/me", middleware.AuthRequired(db), authH.Me)
	api.POST("/auth/change-password", middleware.AuthRequired(db), authH.ChangePassword)
	api.GET("/auth/security", middleware.AuthRequired(db), accH.Security)
	api.POST("/auth/username", middleware.AuthRequired(db), accH.ChangeUsername)
	api.POST("/auth/sessions/revoke", middleware.AuthRequired(db), accH.RevokeSessions)
	api.POST("/auth/totp/setup", middleware.AuthRequired(db), totpH.Setup)
	api.POST("/auth/totp/enable", middleware.AuthRequired(db), totpH.Enable)
	api.POST("/auth/totp/disable", middleware.AuthRequired(db), totpH.Disable)
	api.POST("/auth/passkey/register/begin", middleware.AuthRequired(db), pkH.RegisterBegin)
	api.POST("/auth/passkey/register/finish", middleware.AuthRequired(db), pkH.RegisterFinish)
	api.DELETE("/auth/passkey/:id", middleware.AuthRequired(db), pkH.Remove)

	// 需认证接口
	auth := api.Group("", middleware.AuthRequired(db))

	dashH := handler.NewDashboardHandler(db)
	auth.GET("/dashboard/summary", dashH.Summary)

	qsoH := handler.NewQsoHandler(db)
	auth.GET("/qso-records", qsoH.List)
	auth.POST("/qso-records", qsoH.Create)
	auth.POST("/qso-records/import", qsoH.ImportADIF)
	auth.GET("/qso-records/:id", qsoH.Get)
	auth.PUT("/qso-records/:id", qsoH.Update)
	auth.DELETE("/qso-records/:id", qsoH.Delete)

	cardH := handler.NewCardHandler(db)
	auth.GET("/card-records", cardH.List)
	auth.POST("/card-records", cardH.Create)
	auth.GET("/card-records/:id", cardH.Get)
	auth.PUT("/card-records/:id", cardH.Update)
	auth.DELETE("/card-records/:id", cardH.Delete)
	auth.POST("/card-records/:id/issue", cardH.Issue)
	auth.POST("/card-records/:id/sent", cardH.ConfirmSent)
	auth.POST("/card-records/:id/received", cardH.ConfirmReceived)
	auth.POST("/card-records/:id/resend", cardH.Resend)
	auth.GET("/card-records/:id/tracking", cardH.GetTracking)
	auth.POST("/card-records/:id/tracking", cardH.UpdateTracking)
	auth.POST("/card-records/:id/send-mail", cardH.SendMail)
	auth.POST("/card-records/:id/return-toggle", cardH.ToggleReturnMail)
	auth.POST("/card-records/:id/return-receive", cardH.ReceiveReturnMail)
	auth.POST("/card-records/batch-tracking-update", cardH.BatchTrackingUpdate)
	auth.POST("/card-records/from-qso", cardH.FromQso)
	auth.POST("/upload/image", handler.UploadImage)

	exH := handler.NewExchangeHandler(db)
	expH := handler.NewExportHandler(db)
	auth.GET("/exchange/online/requests", exH.ListRequests)
	auth.POST("/exchange/online/requests", exH.CreateRequest)
	auth.POST("/exchange/online/requests/:id/approve", exH.Approve)
	auth.POST("/exchange/online/requests/:id/reject", exH.Reject)
	auth.POST("/exchange/online/requests/:id/create-card", exH.CreateCard)
	auth.POST("/exchange/online/requests/:id/send-address", exH.SendAddress)
	auth.POST("/exchange/online/requests/:id/receive-return", exH.ReceiveReturnCard)

	// 全量备份与覆盖式导入：拿到就能接管整个后台，所以要求二次验证
	auth.GET("/admin/export", middleware.RequireMFA(db, cfg.RequireMFA), expH.All)
	auth.POST("/admin/import", middleware.RequireMFA(db, cfg.RequireMFA), expH.Import)

	auth.GET("/exchange/offline/activities", exH.ListActivities)
	auth.POST("/exchange/offline/activities", exH.CreateActivity)
	auth.PUT("/exchange/offline/activities/:id", exH.UpdateActivity)

	recvH := handler.NewReceiveHandler(db)
	auth.GET("/receive-records", recvH.List)
	auth.POST("/receive-records", recvH.Create)

	addrH := handler.NewAddressHandler(db)
	auth.GET("/address/book", addrH.ListBook)
	auth.POST("/address/book", addrH.CreateBook)
	auth.PUT("/address/book/:id", addrH.UpdateBook)
	auth.DELETE("/address/book/:id", addrH.DeleteBook)
	auth.GET("/address/bureaus", addrH.ListBureaus)
	auth.POST("/address/bureaus", addrH.CreateBureau)
	auth.PUT("/address/bureaus/:id", addrH.UpdateBureau)
	auth.DELETE("/address/bureaus/:id", addrH.DeleteBureau)

	stH := handler.NewStationHandler(db)
	auth.GET("/station/profile", stH.GetProfile)
	auth.POST("/station/profile", stH.CreateProfile)
	auth.PUT("/station/profile/:id", stH.UpdateProfile)
	auth.GET("/station/cards", stH.ListCards)
	auth.POST("/station/cards", stH.CreateCard)
	auth.PUT("/station/cards/:id", stH.UpdateCard)
	auth.GET("/station/equipments", stH.ListEquipments)
	auth.POST("/station/equipments", stH.CreateEquipment)

	// SMTP 与快递查询设置里含凭据/密钥，同样要求二次验证
	setH := handler.NewSettingsHandler(db)
	auth.GET("/settings/smtp", middleware.RequireMFA(db, cfg.RequireMFA), setH.GetSmtp)
	auth.POST("/settings/smtp", middleware.RequireMFA(db, cfg.RequireMFA), setH.SaveSmtp)
	auth.POST("/settings/smtp/test", middleware.RequireMFA(db, cfg.RequireMFA), setH.TestSmtp)
	auth.GET("/settings/tracking", middleware.RequireMFA(db, cfg.RequireMFA), setH.GetTracking)
	auth.POST("/settings/tracking", middleware.RequireMFA(db, cfg.RequireMFA), setH.SaveTracking)
	auth.GET("/settings/site", setH.GetSite)
	auth.POST("/settings/site", setH.SaveSite)

	// 插件系统端点与功能插件路由：/api/plugins（列表/开关）、/api/ext/<name>/...
	pm.Mount(api)

	log.Printf("QSL 管理系统启动 http://localhost:%s", cfg.ServerPort)
	log.Printf("数据库: %s", cfg.DBDriver)
	if cfg.DBDriver == "sqlite" {
		log.Printf("SQLite 文件: %s", cfg.DBPath)
	}

	r.Run(":" + cfg.ServerPort)
}

// normalizeLegacySceneTypes 一次性数据迁移：ONLINE_EYEBALL -> EYEBALL。
//
// 四条语句各自先探一行，命中才真正 UPDATE。迁移做完之后每次启动都是纯读，
// 少掉 4 次全表写（SQLite 上会直接产生 WAL 帧）。
func normalizeLegacySceneTypes(db *gorm.DB) {
	type job struct {
		label  string
		model  interface{}
		column string
	}
	jobs := []job{
		{"exchange_requests.scene_type", &model.ExchangeRequest{}, "scene_type"},
		{"card_records.scene_type", &model.CardRecord{}, "scene_type"},
		{"card_records.card_type", &model.CardRecord{}, "card_type"},
		{"receive_records.business_type", &model.ReceiveRecord{}, "business_type"},
	}
	for _, j := range jobs {
		var hit int64
		if err := db.Model(j.model).Where(j.column+" = ?", "ONLINE_EYEBALL").Limit(1).Count(&hit).Error; err != nil || hit == 0 {
			continue
		}
		res := db.Model(j.model).Where(j.column+" = ?", "ONLINE_EYEBALL").Update(j.column, "EYEBALL")
		if res.Error != nil {
			log.Printf("[migrate] %s 归一化失败: %v", j.label, res.Error)
			continue
		}
		log.Printf("[migrate] %s: %d 行 ONLINE_EYEBALL -> EYEBALL", j.label, res.RowsAffected)
	}
}

// bootstrapAdmin 首次启动时建管理员账号。
//
// 没配 ADMIN_PASS 就随机生成一个并打印一次——以前默认是 admin123，
// 公网上等于没有口令，爆破字典第一条就是它。
func bootstrapAdmin(cfg *config.Config, db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Where("username = ?", cfg.AdminUser).Count(&count)
	if count > 0 {
		return
	}

	pass := cfg.AdminPass
	generated := pass == ""
	if generated {
		pass = randomPassword(20)
	}
	hash, _ := middleware.HashPassword(pass)
	db.Create(&model.User{
		Username:     cfg.AdminUser,
		HashPassword: hash,
		DisplayName:  "管理员",
		IsAdmin:      true,
		IsActive:     true,
	})
	log.Printf("已创建默认管理员: %s", cfg.AdminUser)

	if generated {
		log.Printf("======================================================")
		log.Printf(" 初始口令（只打印这一次，登录后立刻改掉）: %s", pass)
		log.Printf("======================================================")
	}
}

// escapeHatch 逃生开关：ADMIN_RESET=1 启动时清空管理员的第二因子、失败计数，
// 并重置成一个一次性口令。丢了手机、丢了恢复码、忘了口令，这是唯一还能进门的路。
//
// 用法：docker compose run --rm -e ADMIN_RESET=1 qsl   （或临时改 compose 环境变量后重启）
func escapeHatch(cfg *config.Config, db *gorm.DB) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("ADMIN_RESET")), "1") {
		return
	}

	var user model.User
	if err := db.Where("username = ?", cfg.AdminUser).First(&user).Error; err != nil {
		log.Printf("[逃生] 账号 %s 不存在，跳过重置", cfg.AdminUser)
		return
	}

	pass := randomPassword(20)
	hash, _ := middleware.HashPassword(pass)
	if err := handler.ResetSecondFactor(db, cfg.AdminUser); err != nil {
		log.Printf("[逃生] 清空第二因子失败: %v", err)
		return
	}
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("hash_password", hash).Error; err != nil {
		log.Printf("[逃生] 重置口令失败: %v", err)
		return
	}

	log.Printf("======================================================")
	log.Printf("[逃生] 已清空 %s 的全部通行密钥与二步验证", cfg.AdminUser)
	log.Printf("[逃生] 一次性口令: %s", pass)
	log.Printf("[逃生] 登录后请立刻改密并重新绑定，然后把 ADMIN_RESET 去掉")
	log.Printf("======================================================")
	log.Printf("[逃生] 记得移除 ADMIN_RESET=1，否则每次重启都会重置")
}

// newWebAuthn 初始化通行密钥。没配域名就返回 nil，整个功能关闭，其余照常。
func newWebAuthn(cfg *config.Config) *webauthn.WebAuthn {
	if cfg.WebAuthnRPID == "" || len(cfg.WebAuthnOrigins) == 0 {
		log.Printf("[passkey] 未配置 WEBAUTHN_RP_ID / WEBAUTHN_ORIGINS，通行密钥功能关闭")
		return nil
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "QSL 管理台",
		RPID:          cfg.WebAuthnRPID,
		RPOrigins:     cfg.WebAuthnOrigins,
		// 要支持"免用户名直接登录"，必须要求 resident key + 用户验证
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
		AttestationPreference: protocol.PreferNoAttestation,
	})
	if err != nil {
		log.Printf("[passkey] 初始化失败，功能关闭: %v", err)
		return nil
	}
	log.Printf("[passkey] 已启用 RP ID=%s 来源=%v", cfg.WebAuthnRPID, cfg.WebAuthnOrigins)
	return wa
}

// randomPassword 生成不易混淆的随机口令（去掉了 0/O/1/I/l）。
func randomPassword(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "ChangeMe-Please-Now"
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}
