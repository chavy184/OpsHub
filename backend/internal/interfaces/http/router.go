package http

import (
	"net/http"
	"os"
	"path/filepath"

	appAlert "ops-hub/internal/application/alert"
	appArchive "ops-hub/internal/application/archive"
	appBackup "ops-hub/internal/application/backup"
	appContainer "ops-hub/internal/application/container"
	appCredential "ops-hub/internal/application/credential"
	appDocument "ops-hub/internal/application/document"
	appHost "ops-hub/internal/application/host"
	appImageSync "ops-hub/internal/application/image_sync"
	appNotification "ops-hub/internal/application/notification"
	appRelease "ops-hub/internal/application/release"
	appService "ops-hub/internal/application/service"
	appSetting "ops-hub/internal/application/setting"
	"ops-hub/internal/config"
	infraBackup "ops-hub/internal/infrastructure/backup"
	"ops-hub/internal/infrastructure/jenkins"
	"ops-hub/internal/infrastructure/metrics"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有 HTTP 路由
func RegisterRoutes(
	r *gin.Engine,
	serviceUC *appService.UseCase,
	releaseUC *appRelease.UseCase,
	credentialUC *appCredential.UseCase,
	hostUC *appHost.UseCase,
	settingUC *appSetting.UseCase,
	collector *metrics.Collector,
	alertUC *appAlert.UseCase,
	notificationUC *appNotification.UseCase,
	jenkinsClient *jenkins.Client,
	containerUC *appContainer.UseCase,
	imageSyncUC *appImageSync.UseCase,
	backupUC *appBackup.UseCase,
	backupScheduler *infraBackup.Scheduler,
	migrationExecutor *infraBackup.MigrationExecutor,
	objectSyncExecutor *infraBackup.ObjectSyncExecutor,
	archiveUC *appArchive.UseCase,
	securityCfg config.SecurityConfig,
	staticDir string,
	documentCfg config.DocumentConfig,
) {
	// 健康检查
	healthHandler := func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	}
	r.GET("/health", healthHandler)
	r.HEAD("/health", healthHandler)

	api := r.Group("/api/v1")
	authHandler := NewAuthHandler(securityCfg.AuthSecret, securityCfg.AdminUsername, securityCfg.AdminPassword)
	authHandler.RegisterPublicRoutes(api)
	api.Use(AuthMiddleware(authHandler))
	authHandler.RegisterProtectedRoutes(api)

	// 服务台账
	svcHandler := NewServiceHandler(serviceUC)
	svcHandler.RegisterRoutes(api)

	// 发布与回滚
	relHandler := NewReleaseHandler(releaseUC)
	relHandler.RegisterRoutes(api)

	// 凭证管理
	credHandler := NewCredentialHandler(credentialUC)
	credHandler.RegisterRoutes(api)

	// 机器管理
	hostHandler := NewHostHandler(hostUC, collector)
	hostHandler.RegisterRoutes(api)

	// 系统设置
	settingHandler := NewSettingHandler(settingUC)
	settingHandler.RegisterRoutes(api)

	// 日志代理（Loki / file / docker）
	logHandler := NewLogHandler(serviceUC, hostUC)
	logHandler.RegisterRoutes(api)

	// 告警中心
	alertHandler := NewAlertHandler(alertUC)
	alertHandler.RegisterRoutes(api)

	// 通知推送
	notificationHandler := NewNotificationHandler(notificationUC)
	notificationHandler.RegisterRoutes(api)

	// Jenkins 代理
	if jenkinsClient != nil {
		jenkinsHandler := NewJenkinsHandler(jenkinsClient, settingUC)
		jenkinsHandler.RegisterRoutes(api)
	}

	// 全局聚合搜索（命令面板）
	searchHandler := NewSearchHandler(serviceUC, hostUC)
	searchHandler.RegisterRoutes(api)

	// 容器管理
	containerHandler := NewContainerHandler(containerUC)
	containerHandler.RegisterRoutes(api)

	// 镜像同步
	imageSyncHandler := NewImageSyncHandler(imageSyncUC)
	imageSyncHandler.RegisterRoutes(api)

	// 数据备份
	backupHandler := NewBackupHandler(backupUC, backupScheduler, migrationExecutor, objectSyncExecutor)
	backupHandler.RegisterRoutes(api)

	// 文档管理
	documentUC := appDocument.NewUseCase(documentCfg.BaseDir, documentCfg.MaxFileSize)
	documentHandler := NewDocumentHandler(documentUC)
	documentHandler.RegisterRoutes(api)

	// 解压目录
	archiveHandler := NewArchiveHandler(archiveUC)
	archiveHandler.RegisterRoutes(api)

	// 前端静态文件 (SPA)
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		r.Use(serveSPA(staticDir))
	}
}

// serveSPA 提供 SPA 静态文件服务，未匹配的路径回退到 index.html
func serveSPA(staticDir string) gin.HandlerFunc {
	fs := http.Dir(staticDir)
	fileServer := http.FileServer(fs)
	return func(c *gin.Context) {
		// 跳过 API 和健康检查路径
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.Next()
			return
		}
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		// 尝试提供静态文件
		filePath := filepath.Join(staticDir, c.Request.URL.Path)
		if _, err := os.Stat(filePath); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}
		// SPA 回退：返回 index.html
		c.File(filepath.Join(staticDir, "index.html"))
		c.Abort()
	}
}
