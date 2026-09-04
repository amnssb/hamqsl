package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"qsl-management/internal/config"
	"qsl-management/internal/handler"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
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
		db, err = gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	}

	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	db.AutoMigrate(
		&model.User{}, &model.StationProfile{}, &model.StationCard{}, &model.StationEquipment{},
		&model.QsoRecord{}, &model.CardRecord{}, &model.AddressBookEntry{}, &model.BureauEntry{},
		&model.ExchangeRequest{}, &model.OfflineActivity{}, &model.ReceiveRecord{},
		&model.SystemSetting{}, &model.AuditLog{},
	)

	// 历史数据归一化：老版本场景类型 ONLINE_EYEBALL -> EYEBALL
	db.Model(&model.ExchangeRequest{}).Where("scene_type = ?", "ONLINE_EYEBALL").Update("scene_type", "EYEBALL")
	db.Model(&model.CardRecord{}).Where("scene_type = ?", "ONLINE_EYEBALL").Update("scene_type", "EYEBALL")
	db.Model(&model.CardRecord{}).Where("card_type = ?", "ONLINE_EYEBALL").Update("card_type", "EYEBALL")
	db.Model(&model.ReceiveRecord{}).Where("business_type = ?", "ONLINE_EYEBALL").Update("business_type", "EYEBALL")

	var count int64
	db.Model(&model.User{}).Where("username = ?", cfg.AdminUser).Count(&count)
	if count == 0 {
		hash, _ := middleware.HashPassword(cfg.AdminPass)
		db.Create(&model.User{Username: cfg.AdminUser, HashPassword: hash, DisplayName: "管理员", IsAdmin: true})
		log.Printf("已创建默认管理员: %s", cfg.AdminUser)
	}

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

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
			p := filepath.Join(webDir, filepath.Clean("/"+c.Request.URL.Path))
			if st, serr := os.Stat(p); serr == nil && !st.IsDir() {
				c.File(p)
				return
			}
			if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[:5] == "/api/" {
				c.JSON(404, gin.H{"code": "QSL-404", "message": "接口不存在"})
				return
			}
			c.File(filepath.Join(webDir, "index.html"))
		})
		log.Printf("前端静态目录: %s（单进程托管）", webDir)
	}

	r.GET("/", func(c *gin.Context) {
		if serveWeb {
			// 托管前端时根路径返回门户页面，而不是 API 说明
			c.File(filepath.Join(webDir, "index.html"))
			return
		}
		c.JSON(200, gin.H{"message": "QSL 卡片管理系统 API", "version": "1.0.0", "db": cfg.DBDriver})
	})

	api := r.Group("/api")

	// 公开接口（无需登录）
	pub := api.Group("/public")
	pubH := handler.NewPublicHandler(db)
	pub.GET("/site-info", pubH.SiteInfo)
	pub.GET("/stats", pubH.PublicStats)
	pub.GET("/station-cards", pubH.StationCards)
	pub.GET("/station-mail-info", pubH.StationMailInfo)
	pub.GET("/exchange-status/:request_code", pubH.ExchangeStatus)
	pub.GET("/bureaus", pubH.Bureaus)
	pub.POST("/exchange-online", pubH.SubmitExchange)      // 线上换卡申请
	pub.POST("/confirm-receipt", pubH.ConfirmReceipt)      // 确认收件
	pub.POST("/return-mail", pubH.ReturnMail)              // 回寄登记：对方确认收卡后登记回寄方式与单号
	pub.POST("/exchange-return-mail", pubH.SubmitReturnMail) // SWL 反寄登记
	pub.GET("/cards/:card_code", pubH.GetCardByCode)       // 按编号查卡片（公开）
	pub.GET("/tracking", pubH.QueryTracking)               // 快递追踪
	pub.GET("/tracking/:tracking_number", pubH.QueryTracking)

	// 认证
	authH := handler.NewAuthHandler(db)
	api.POST("/auth/login", authH.Login)
	api.GET("/auth/me", middleware.AuthRequired(), authH.Me)
	api.POST("/auth/change-password", middleware.AuthRequired(), authH.ChangePassword)

	// 需认证接口
	auth := api.Group("", middleware.AuthRequired())

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
	auth.GET("/admin/export", expH.All)
	auth.POST("/admin/import", expH.Import)
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

	setH := handler.NewSettingsHandler(db)
	auth.GET("/settings/smtp", setH.GetSmtp)
	auth.POST("/settings/smtp", setH.SaveSmtp)
	auth.POST("/settings/smtp/test", setH.TestSmtp)
	auth.GET("/settings/tracking", setH.GetTracking)
	auth.POST("/settings/tracking", setH.SaveTracking)
	auth.GET("/settings/site", setH.GetSite)
	auth.POST("/settings/site", setH.SaveSite)

	log.Printf("QSL 管理系统启动 http://localhost:%s", cfg.ServerPort)
	log.Printf("数据库: %s", cfg.DBDriver)
	if cfg.DBDriver == "sqlite" {
		log.Printf("SQLite 文件: %s", cfg.DBPath)
	}

	r.Run(":" + cfg.ServerPort)
}
