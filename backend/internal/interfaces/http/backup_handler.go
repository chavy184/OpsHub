package http

import (
	appBackup "ops-hub/internal/application/backup"
	domainBackup "ops-hub/internal/domain/backup"
	infraBackup "ops-hub/internal/infrastructure/backup"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// backup 领域错误 → AppError 映射
var backupErrMappings = []apperr.DomainErrorMapping{
	{Target: domainBackup.ErrTaskNotFound, AppErr: apperr.NotFound("备份任务不存在")},
	{Target: domainBackup.ErrRecordNotFound, AppErr: apperr.NotFound("备份记录不存在")},
	{Target: domainBackup.ErrTaskNameExists, AppErr: apperr.Conflict("备份任务名称已存在")},
	{Target: domainBackup.ErrInvalidCronExpr, AppErr: apperr.BadRequest("无效的 cron 表达式")},
	{Target: domainBackup.ErrTaskDisabled, AppErr: apperr.BadRequest("备份任务已禁用")},
	{Target: domainBackup.ErrMigrationTaskNotFound, AppErr: apperr.NotFound("迁移任务不存在")},
	{Target: domainBackup.ErrMigrationRecordNotFound, AppErr: apperr.NotFound("迁移记录不存在")},
	{Target: domainBackup.ErrInvalidMigrationMode, AppErr: apperr.BadRequest("无效的迁移模式")},
	{Target: domainBackup.ErrInvalidMigrationDBType, AppErr: apperr.BadRequest("无效的迁移数据库类型")},
	{Target: domainBackup.ErrMigrationDBNamesRequired, AppErr: apperr.BadRequest("迁移数据库列表不能为空")},
	{Target: domainBackup.ErrOverwriteConfirmRequired, AppErr: apperr.BadRequest("覆盖迁移需要二次确认")},
	{Target: domainBackup.ErrObjectSyncTaskNotFound, AppErr: apperr.NotFound("对象同步任务不存在")},
	{Target: domainBackup.ErrObjectSyncRecordNotFound, AppErr: apperr.NotFound("对象同步记录不存在")},
	{Target: domainBackup.ErrInvalidObjectSyncMode, AppErr: apperr.BadRequest("无效的对象同步模式")},
}

type BackupHandler struct {
	uc                 *appBackup.UseCase
	scheduler          *infraBackup.Scheduler
	migrationExecutor  *infraBackup.MigrationExecutor
	objectSyncExecutor *infraBackup.ObjectSyncExecutor
}

func NewBackupHandler(uc *appBackup.UseCase, scheduler *infraBackup.Scheduler, migrationExecutor *infraBackup.MigrationExecutor, objectSyncExecutor *infraBackup.ObjectSyncExecutor) *BackupHandler {
	return &BackupHandler{uc: uc, scheduler: scheduler, migrationExecutor: migrationExecutor, objectSyncExecutor: objectSyncExecutor}
}

func (h *BackupHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/backup")
	{
		g.POST("/tasks", apperr.Handle(h.CreateTask))
		g.PUT("/tasks/:id", apperr.Handle(h.UpdateTask))
		g.GET("/tasks", apperr.Handle(h.ListTasks))
		g.GET("/tasks/:id", apperr.Handle(h.GetTask))
		g.DELETE("/tasks/:id", apperr.Handle(h.DeleteTask))
		g.POST("/tasks/:id/trigger", apperr.Handle(h.TriggerTask))
		g.GET("/records", apperr.Handle(h.ListRecords))

		g.POST("/migrations", apperr.Handle(h.CreateMigrationTask))
		g.PUT("/migrations/:id", apperr.Handle(h.UpdateMigrationTask))
		g.GET("/migrations", apperr.Handle(h.ListMigrationTasks))
		g.GET("/migrations/:id", apperr.Handle(h.GetMigrationTask))
		g.DELETE("/migrations/:id", apperr.Handle(h.DeleteMigrationTask))
		g.POST("/migrations/:id/execute", apperr.Handle(h.ExecuteMigrationTask))
		g.GET("/migration-records", apperr.Handle(h.ListMigrationRecords))
		g.GET("/migration-records/:id", apperr.Handle(h.GetMigrationRecord))

		g.POST("/object-sync/tasks", apperr.Handle(h.CreateObjectSyncTask))
		g.PUT("/object-sync/tasks/:id", apperr.Handle(h.UpdateObjectSyncTask))
		g.GET("/object-sync/tasks", apperr.Handle(h.ListObjectSyncTasks))
		g.GET("/object-sync/tasks/:id", apperr.Handle(h.GetObjectSyncTask))
		g.DELETE("/object-sync/tasks/:id", apperr.Handle(h.DeleteObjectSyncTask))
		g.POST("/object-sync/tasks/:id/execute", apperr.Handle(h.ExecuteObjectSyncTask))
		g.GET("/object-sync/records", apperr.Handle(h.ListObjectSyncRecords))
		g.GET("/object-sync/records/:id", apperr.Handle(h.GetObjectSyncRecord))
	}
}

// CreateTask 创建备份任务
func (h *BackupHandler) CreateTask(c *gin.Context) error {
	var cmd appBackup.CreateTaskCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dto, err := h.uc.CreateTask(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}

	h.scheduler.Reload()
	response.Created(c, dto)
	return nil
}

// UpdateTask 更新备份任务
func (h *BackupHandler) UpdateTask(c *gin.Context) error {
	var cmd appBackup.UpdateTaskCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")

	dto, err := h.uc.UpdateTask(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}

	h.scheduler.Reload()
	response.OK(c, dto)
	return nil
}

// ListTasks 查询备份任务列表
func (h *BackupHandler) ListTasks(c *gin.Context) error {
	var cmd appBackup.TaskQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dtos, total, err := h.uc.ListTasks(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}

	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

// GetTask 获取单个备份任务
func (h *BackupHandler) GetTask(c *gin.Context) error {
	dto, err := h.uc.GetTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// DeleteTask 删除备份任务
func (h *BackupHandler) DeleteTask(c *gin.Context) error {
	if err := h.uc.DeleteTask(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	h.scheduler.Reload()
	response.OK(c, nil)
	return nil
}

// TriggerTask 手动触发备份
func (h *BackupHandler) TriggerTask(c *gin.Context) error {
	if err := h.scheduler.TriggerNow(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, gin.H{"message": "备份任务已触发"})
	return nil
}

// ListRecords 查询备份记录
func (h *BackupHandler) ListRecords(c *gin.Context) error {
	var cmd appBackup.RecordQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dtos, total, err := h.uc.ListRecords(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}

	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

// CreateMigrationTask 创建数据库迁移任务
func (h *BackupHandler) CreateMigrationTask(c *gin.Context) error {
	var cmd appBackup.CreateMigrationTaskCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dto, err := h.uc.CreateMigrationTask(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.Created(c, dto)
	return nil
}

// UpdateMigrationTask 更新数据库迁移任务
func (h *BackupHandler) UpdateMigrationTask(c *gin.Context) error {
	var cmd appBackup.UpdateMigrationTaskCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")

	dto, err := h.uc.UpdateMigrationTask(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// ListMigrationTasks 查询数据库迁移任务列表
func (h *BackupHandler) ListMigrationTasks(c *gin.Context) error {
	var cmd appBackup.MigrationTaskQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dtos, total, err := h.uc.ListMigrationTasks(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

// GetMigrationTask 获取单个数据库迁移任务
func (h *BackupHandler) GetMigrationTask(c *gin.Context) error {
	dto, err := h.uc.GetMigrationTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// DeleteMigrationTask 删除数据库迁移任务
func (h *BackupHandler) DeleteMigrationTask(c *gin.Context) error {
	if err := h.uc.DeleteMigrationTask(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, nil)
	return nil
}

// ExecuteMigrationTask 手动执行数据库迁移任务
func (h *BackupHandler) ExecuteMigrationTask(c *gin.Context) error {
	var cmd appBackup.ExecuteMigrationCmd
	_ = c.ShouldBindJSON(&cmd)

	if err := h.migrationExecutor.ExecuteNow(c.Request.Context(), c.Param("id"), cmd.ConfirmOverwrite); err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, gin.H{"message": "迁移任务已触发"})
	return nil
}

// ListMigrationRecords 查询数据库迁移执行记录
func (h *BackupHandler) ListMigrationRecords(c *gin.Context) error {
	var cmd appBackup.MigrationRecordQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dtos, total, err := h.uc.ListMigrationRecords(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

// GetMigrationRecord 获取数据库迁移记录详情
func (h *BackupHandler) GetMigrationRecord(c *gin.Context) error {
	dto, err := h.uc.GetMigrationRecord(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// CreateObjectSyncTask 创建对象存储同步任务
func (h *BackupHandler) CreateObjectSyncTask(c *gin.Context) error {
	var cmd appBackup.CreateObjectSyncTaskCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.CreateObjectSyncTask(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.Created(c, dto)
	return nil
}

// UpdateObjectSyncTask 更新对象存储同步任务
func (h *BackupHandler) UpdateObjectSyncTask(c *gin.Context) error {
	var cmd appBackup.UpdateObjectSyncTaskCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")
	dto, err := h.uc.UpdateObjectSyncTask(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// ListObjectSyncTasks 查询对象存储同步任务
func (h *BackupHandler) ListObjectSyncTasks(c *gin.Context) error {
	var cmd appBackup.ObjectSyncTaskQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dtos, total, err := h.uc.ListObjectSyncTasks(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

// GetObjectSyncTask 获取对象存储同步任务
func (h *BackupHandler) GetObjectSyncTask(c *gin.Context) error {
	dto, err := h.uc.GetObjectSyncTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// DeleteObjectSyncTask 删除对象存储同步任务
func (h *BackupHandler) DeleteObjectSyncTask(c *gin.Context) error {
	if err := h.uc.DeleteObjectSyncTask(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, nil)
	return nil
}

// ExecuteObjectSyncTask 手动执行对象存储同步任务
func (h *BackupHandler) ExecuteObjectSyncTask(c *gin.Context) error {
	var cmd appBackup.ExecuteObjectSyncCmd
	_ = c.ShouldBindJSON(&cmd)
	if err := h.objectSyncExecutor.ExecuteNow(c.Request.Context(), c.Param("id"), cmd.ConfirmOverwrite); err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, gin.H{"message": "对象同步任务已触发"})
	return nil
}

// ListObjectSyncRecords 查询对象存储同步记录
func (h *BackupHandler) ListObjectSyncRecords(c *gin.Context) error {
	var cmd appBackup.ObjectSyncRecordQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dtos, total, err := h.uc.ListObjectSyncRecords(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

// GetObjectSyncRecord 获取对象存储同步记录详情
func (h *BackupHandler) GetObjectSyncRecord(c *gin.Context) error {
	dto, err := h.uc.GetObjectSyncRecord(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, backupErrMappings)
	}
	response.OK(c, dto)
	return nil
}
