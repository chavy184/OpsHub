package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	appAlert "ops-hub/internal/application/alert"
	appArchive "ops-hub/internal/application/archive"
	appBackup "ops-hub/internal/application/backup"
	appContainer "ops-hub/internal/application/container"
	appCredential "ops-hub/internal/application/credential"
	appHost "ops-hub/internal/application/host"
	appImageSync "ops-hub/internal/application/image_sync"
	appNotification "ops-hub/internal/application/notification"
	appRelease "ops-hub/internal/application/release"
	appService "ops-hub/internal/application/service"
	appSetting "ops-hub/internal/application/setting"
	"ops-hub/internal/config"
	"ops-hub/internal/domain/notification"
	domainSetting "ops-hub/internal/domain/setting"
	infraArchive "ops-hub/internal/infrastructure/archive"
	infraBackup "ops-hub/internal/infrastructure/backup"
	"ops-hub/internal/infrastructure/crypto"
	"ops-hub/internal/infrastructure/executor"
	"ops-hub/internal/infrastructure/healthcheck"
	"ops-hub/internal/infrastructure/jenkins"
	"ops-hub/internal/infrastructure/metrics"
	"ops-hub/internal/infrastructure/persistence"
	handler "ops-hub/internal/interfaces/http"
	appLogger "ops-hub/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	// 加载配置文件
	cfgPath := os.Getenv("OPSHUB_CONFIG")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	// 初始化日志
	zapLogger := appLogger.Init(cfg.Log)
	defer zapLogger.Sync()
	// 数据库连接
	db := initDB(cfg.Database)

	// 自动迁移
	autoMigrate(db)

	// 加密器（AES-256-GCM）
	encryptor := crypto.NewAESEncryptor(cfg.Security.EncryptKey)

	// Repository
	serviceRepo := persistence.NewGormServiceRepository(db)
	envRepo := persistence.NewGormServiceEnvRepository(db)
	releaseRepo := persistence.NewGormReleaseRecordRepository(db)
	stepRepo := persistence.NewReleaseStepLogRepository(db)
	credRepo := persistence.NewCredentialRepository(db)
	hostRepo := persistence.NewHostRepository(db)
	settingRepo := persistence.NewSettingRepository(db)
	metricRepo := persistence.NewHostMetricRepo(db)
	alertRepo := persistence.NewAlertEventRepository(db)
	notifChannelRepo := persistence.NewNotificationChannelRepository(db)
	notifRuleRepo := persistence.NewNotificationRuleRepository(db)
	notifLogRepo := persistence.NewNotificationLogRepository(db)
	containerRepo := persistence.NewContainerRepository(db)
	imageSyncRecordRepo := persistence.NewImageSyncRecordRepository(db)
	backupTaskRepo := persistence.NewBackupTaskRepository(db)
	backupRecordRepo := persistence.NewBackupRecordRepository(db)
	migrationTaskRepo := persistence.NewMigrationTaskRepository(db)
	migrationRecordRepo := persistence.NewMigrationRecordRepository(db)
	migrationRecordItemRepo := persistence.NewMigrationRecordItemRepository(db)
	objectSyncTaskRepo := persistence.NewObjectSyncTaskRepository(db)
	objectSyncRecordRepo := persistence.NewObjectSyncRecordRepository(db)
	objectSyncRecordItemRepo := persistence.NewObjectSyncRecordItemRepository(db)

	// 执行器（MVP 使用 SSH）
	execCfg := executor.DefaultExecutorConfig()
	sshExecutor := executor.NewSSHExecutor(execCfg)

	// Jenkins 客户端（从配置初始化，设置页面可覆盖）
	jenkinsClient := jenkins.NewClient(cfg.Jenkins.URL, cfg.Jenkins.User, cfg.Jenkins.Token)
	jenkinsExec := executor.NewJenkinsExecutor(jenkinsClient)
	jenkinsExec.SetSettingRepo(settingRepo)

	// Application UseCase
	serviceUC := appService.NewUseCase(serviceRepo, envRepo)
	releaseUC := appRelease.NewUseCase(releaseRepo, stepRepo, serviceRepo, envRepo, hostRepo, sshExecutor)
	releaseUC.SetJenkinsExecutor(jenkinsExec)
	releaseUC.SetAdminPassword(cfg.Security.AdminPassword)
	credentialUC := appCredential.NewUseCase(credRepo, encryptor)
	hostUC := appHost.NewUseCase(hostRepo, credRepo, encryptor, metricRepo)
	containerUC := appContainer.NewUseCase(containerRepo, hostUC)
	imageSyncUC := appImageSync.NewUseCase(imageSyncRecordRepo, hostUC)
	settingUC := appSetting.NewUseCase(settingRepo)
	alertUC := appAlert.NewUseCase(alertRepo)
	notificationUC := appNotification.NewUseCase(notifChannelRepo, notifRuleRepo, notifLogRepo)
	archiveUC := appArchive.NewUseCase(infraArchive.NewDebParser())

	// 发布通知接线
	releaseUC.SetNotifier(notificationUC)

	// 备份模块
	backupUC := appBackup.NewUseCase(backupTaskRepo, backupRecordRepo, migrationTaskRepo, migrationRecordRepo, migrationRecordItemRepo, objectSyncTaskRepo, objectSyncRecordRepo, objectSyncRecordItemRepo, encryptor)
	backupScheduler := infraBackup.NewScheduler(backupUC, hostUC)
	migrationExecutor := infraBackup.NewMigrationExecutor(backupUC)
	objectSyncExecutor := infraBackup.NewObjectSyncExecutor(backupUC)
	if err := backupScheduler.Start(); err != nil {
		log.Printf("[OpsHub] 备份调度器启动失败: %v", err)
	}
	defer backupScheduler.Stop()

	// 种子数据：Jenkins 默认配置
	seedJenkinsDefaults(settingUC, cfg.Jenkins.URL, cfg.Jenkins.User, cfg.Jenkins.Token)

	// 种子数据：资源告警阈值默认值
	seedMetricThresholdDefaults(settingUC)

	// 健康检查调度器
	scheduler := healthcheck.NewScheduler(envRepo)
	scheduler.Start()
	defer scheduler.Stop()

	// 主机指标采集器（SSH 通过 hostUC 的连接池复用）
	metricsCollector := metrics.NewCollector(hostRepo, metricRepo, hostUC)
	// 资源告警聚合器：10s 安静期内的所有告警合并为一条通知，避免多主机刷屏
	alertAgg := newResourceAlertAggregator(notificationUC, 10*time.Second)
	alertAgg.start()
	defer alertAgg.stop()
	metricsCollector.SetAlertCallback(func(hostName, hostAddress, alertType, message string, value float64) {
		alertAgg.push(resourceAlertEvent{
			HostName: hostName, HostAddr: hostAddress,
			AlertType: alertType, Message: message, Value: value, At: time.Now(),
		})
	})
	// 阈值 Provider：每次调用从 settings 读取（带 60s 缓存避免高频 IO）
	metricsCollector.SetThresholdProvider(newSettingThresholdProvider(settingUC))
	metricsCollector.Start()
	defer metricsCollector.Stop()

	// HTTP 路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(zapGinLogger())
	r.Use(corsMiddleware())

	handler.RegisterRoutes(r, serviceUC, releaseUC, credentialUC, hostUC, settingUC, metricsCollector, alertUC, notificationUC, jenkinsClient, containerUC, imageSyncUC, backupUC, backupScheduler, migrationExecutor, objectSyncExecutor, archiveUC, cfg.Security, cfg.Static.Dir, cfg.Document)

	port := fmt.Sprintf("%d", cfg.Server.Port)

	log.Printf("[OpsHub] 服务启动 http://localhost:%s", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func initDB(cfg config.DatabaseConfig) *gorm.DB {
	// GORM 日志输出
	var writer *log.Logger
	if cfg.LogFile != "" {
		dir := filepath.Dir(cfg.LogFile)
		_ = os.MkdirAll(dir, 0755)
		gormLogFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("无法打开 %s，仅使用 stdout: %v", cfg.LogFile, err)
			writer = log.New(os.Stdout, "", log.LstdFlags)
		} else {
			multiWriter := io.MultiWriter(os.Stdout, gormLogFile)
			writer = log.New(multiWriter, "", log.LstdFlags)
		}
	} else {
		writer = log.New(os.Stdout, "", log.LstdFlags)
	}

	gormLog := gormLogger.New(
		writer,
		gormLogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: gormLog,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	return db
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&persistence.ServicePO{},
		&persistence.ServiceEnvPO{},
		&persistence.ReleaseRecordPO{},
		&persistence.TenantPO{},
		&persistence.TenantServiceBindingPO{},
		&persistence.ConfigItemPO{},
		&persistence.ConfigOverridePO{},
		&persistence.AlertEventPO{},
		&persistence.OpAuditLogPO{},
		&persistence.CredentialPO{},
		&persistence.HostPO{},
		&persistence.ReleaseStepLogPO{},
		&persistence.SystemSettingPO{},
		&persistence.HostMetricSnapshotPO{},
		&persistence.NotificationChannelPO{},
		&persistence.NotificationRulePO{},
		&persistence.NotificationLogPO{},
		&persistence.ContainerPO{},
		&persistence.ImageSyncRecordPO{},
		&persistence.BackupTaskPO{},
		&persistence.BackupRecordPO{},
		&persistence.MigrationTaskPO{},
		&persistence.MigrationRecordPO{},
		&persistence.MigrationRecordItemPO{},
		&persistence.ObjectSyncTaskPO{},
		&persistence.ObjectSyncRecordPO{},
		&persistence.ObjectSyncRecordItemPO{},
	)
	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 迁移 SSH 用户名归属：凭证只保存认证材料，主机保存实际登录用户名。
	if e := db.Exec(`
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'credentials' AND column_name = 'username'
	) THEN
		UPDATE hosts h
		SET username = c.username
		FROM credentials c
		WHERE h.credential_id = c.id
		  AND COALESCE(h.username, '') = ''
		  AND COALESCE(c.username, '') <> '';
		ALTER TABLE credentials DROP COLUMN IF EXISTS username;
	END IF;
END $$;
`).Error; e != nil {
		log.Printf("[migrate] move credentials.username to hosts.username 失败: %v", e)
	}

	// 一次性清理 services 表中已下线的冗余字段（IF EXISTS，幂等）
	for _, col := range []string{
		"tech_stack", "runtime_type", "deploy_type", "sla_level", "status",
		"default_branch", "build_command", "start_command", "stop_command",
		"docker_image", "dockerfile_path", "work_dir", "jenkins_job_url",
	} {
		if e := db.Exec("ALTER TABLE services DROP COLUMN IF EXISTS " + col).Error; e != nil {
			log.Printf("[migrate] drop services.%s 失败: %v", col, e)
		}
	}

	// 一次性删除已下线的 service_versions 表（IF EXISTS，幂等）
	if e := db.Exec("DROP TABLE IF EXISTS service_versions").Error; e != nil {
		log.Printf("[migrate] drop table service_versions 失败: %v", e)
	}

	// 修复软删除与唯一索引冲突：改用条件唯一索引（仅对未删除记录生效）
	partialUniqueIndexes := []struct {
		table, oldIdx, newIdx, column string
	}{
		{"credentials", "idx_credentials_name", "uk_credentials_name_active", "name"},
		{"hosts", "idx_hosts_name", "uk_hosts_name_active", "name"},
	}
	for _, idx := range partialUniqueIndexes {
		// 删除 GORM 可能创建的旧普通唯一索引
		db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", idx.oldIdx))
		// 创建条件唯一索引（仅 deleted_at IS NULL 的行参与唯一约束）
		db.Exec(fmt.Sprintf(
			"CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s) WHERE deleted_at IS NULL",
			idx.newIdx, idx.table, idx.column,
		))
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id, Idempotency-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func seedJenkinsDefaults(settingUC *appSetting.UseCase, url, user, token string) {
	defaults := []domainSetting.SystemSetting{
		{SettingKey: "jenkins.url", Value: url, ValueType: "string", Category: "deploy", Description: "Jenkins 服务地址"},
		{SettingKey: "jenkins.user", Value: user, ValueType: "string", Category: "deploy", Description: "Jenkins 用户名"},
		{SettingKey: "jenkins.token", Value: token, ValueType: "string", Category: "deploy", Description: "Jenkins API Token"},
	}
	if err := settingUC.SeedDefaults(context.Background(), defaults); err != nil {
		log.Printf("[OpsHub] Jenkins 默认配置种子失败: %v", err)
	}
}

// 资源告警阈值默认值种子
func seedMetricThresholdDefaults(settingUC *appSetting.UseCase) {
	defaults := []domainSetting.SystemSetting{
		{SettingKey: "metric.threshold.cpu.warning", Value: "90", ValueType: "int", Category: "monitor", Description: "CPU 警告阈值（%）"},
		{SettingKey: "metric.threshold.cpu.critical", Value: "95", ValueType: "int", Category: "monitor", Description: "CPU 严重阈值（%）"},
		{SettingKey: "metric.threshold.mem.warning", Value: "85", ValueType: "int", Category: "monitor", Description: "内存警告阈值（%）"},
		{SettingKey: "metric.threshold.mem.critical", Value: "95", ValueType: "int", Category: "monitor", Description: "内存严重阈值（%）"},
		{SettingKey: "metric.threshold.disk.warning", Value: "85", ValueType: "int", Category: "monitor", Description: "磁盘警告阈值（%）"},
		{SettingKey: "metric.threshold.disk.critical", Value: "95", ValueType: "int", Category: "monitor", Description: "磁盘严重阈值（%）"},
	}
	if err := settingUC.SeedDefaults(context.Background(), defaults); err != nil {
		log.Printf("[OpsHub] 资源阈值默认值种子失败: %v", err)
	}
}

// 从设置中读取阈值的 Provider，带 60s 缓存
func newSettingThresholdProvider(settingUC *appSetting.UseCase) metrics.ThresholdProvider {
	var (
		mu       sync.RWMutex
		cached   metrics.Thresholds
		cachedAt time.Time
		ttl      = 60 * time.Second
	)

	readFloat := func(key string, fallback float64) float64 {
		v, err := settingUC.GetByKey(context.Background(), key)
		if err != nil || v == "" {
			return fallback
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f <= 0 || f > 100 {
			return fallback
		}
		return f
	}

	return func() metrics.Thresholds {
		mu.RLock()
		if time.Since(cachedAt) < ttl && cachedAt != (time.Time{}) {
			defer mu.RUnlock()
			return cached
		}
		mu.RUnlock()

		mu.Lock()
		defer mu.Unlock()
		if time.Since(cachedAt) < ttl && cachedAt != (time.Time{}) {
			return cached
		}
		d := metrics.DefaultThresholds
		cached = metrics.Thresholds{
			CPUWarning:   readFloat("metric.threshold.cpu.warning", d.CPUWarning),
			CPUCritical:  readFloat("metric.threshold.cpu.critical", d.CPUCritical),
			MemWarning:   readFloat("metric.threshold.mem.warning", d.MemWarning),
			MemCritical:  readFloat("metric.threshold.mem.critical", d.MemCritical),
			DiskWarning:  readFloat("metric.threshold.disk.warning", d.DiskWarning),
			DiskCritical: readFloat("metric.threshold.disk.critical", d.DiskCritical),
		}
		cachedAt = time.Now()
		return cached
	}
}

// 资源告警聚合器
// 在指定的安静期内将多个告警合并为一条通知，避免多主机消息刷屏

type resourceAlertEvent struct {
	HostName  string
	HostAddr  string
	AlertType string
	Message   string
	Value     float64
	At        time.Time
}

type resourceAlertAggregator struct {
	notif    *appNotification.UseCase
	debounce time.Duration
	mu       sync.Mutex
	buf      []resourceAlertEvent
	timer    *time.Timer
	stopCh   chan struct{}
}

func newResourceAlertAggregator(notif *appNotification.UseCase, debounce time.Duration) *resourceAlertAggregator {
	return &resourceAlertAggregator{notif: notif, debounce: debounce, stopCh: make(chan struct{})}
}

func (a *resourceAlertAggregator) start() {}

func (a *resourceAlertAggregator) stop() {
	a.mu.Lock()
	if a.timer != nil {
		a.timer.Stop()
	}
	a.mu.Unlock()
	close(a.stopCh)
}

func (a *resourceAlertAggregator) push(ev resourceAlertEvent) {
	a.mu.Lock()
	a.buf = append(a.buf, ev)
	// 重置 debounce 定时器：每收到一个新事件就把 flush 推后 debounce 时长
	if a.timer != nil {
		a.timer.Stop()
	}
	a.timer = time.AfterFunc(a.debounce, a.flush)
	a.mu.Unlock()
}

func (a *resourceAlertAggregator) flush() {
	a.mu.Lock()
	if len(a.buf) == 0 {
		a.mu.Unlock()
		return
	}
	events := a.buf
	a.buf = nil
	a.mu.Unlock()

	title, content := buildAggregatedAlert(events)
	// 临时指纹，避免被去重拦截（聚合报警不走去重字段）
	fingerprint := fmt.Sprintf("agg:%d", time.Now().UnixNano())
	a.notif.Dispatch(context.Background(), notification.EventResourceAlert, fingerprint, title, content)
}

func buildAggregatedAlert(events []resourceAlertEvent) (string, string) {
	// 按主机分组
	type hostBucket struct {
		Name  string
		Addr  string
		Items []resourceAlertEvent
	}
	order := []string{}
	groups := map[string]*hostBucket{}
	for _, ev := range events {
		key := ev.HostAddr + "|" + ev.HostName
		if _, ok := groups[key]; !ok {
			groups[key] = &hostBucket{Name: ev.HostName, Addr: ev.HostAddr}
			order = append(order, key)
		}
		groups[key].Items = append(groups[key].Items, ev)
	}

	hostCount := len(order)
	itemCount := len(events)
	title := fmt.Sprintf("[OpsHub资源] %d 台主机 %d 条告警", hostCount, itemCount)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("概况: %d 台主机 / %d 条告警\n\n", hostCount, itemCount))
	for _, key := range order {
		g := groups[key]
		sb.WriteString(fmt.Sprintf("■ %s (%s)\n", g.Name, g.Addr))
		for _, it := range g.Items {
			sb.WriteString(fmt.Sprintf("  - [%s] %s\n", it.AlertType, it.Message))
		}
	}
	return title, sb.String()
}

// zapGinLogger 用 zap 记录每个 HTTP 请求（替代 gin.Logger）
func zapGinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		if status >= 500 {
			zap.L().Error("request", fields...)
		} else if status >= 400 {
			zap.L().Warn("request", fields...)
		} else {
			zap.L().Info("request", fields...)
		}
	}
}
