package backup

import (
	"context"
	"fmt"
	"log"
	appBackup "ops-hub/internal/application/backup"
	domainBackup "ops-hub/internal/domain/backup"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MigrationExecutor 手动数据库迁移执行器
type MigrationExecutor struct {
	backupUC *appBackup.UseCase
}

func NewMigrationExecutor(backupUC *appBackup.UseCase) *MigrationExecutor {
	return &MigrationExecutor{backupUC: backupUC}
}

func (e *MigrationExecutor) ExecuteNow(ctx context.Context, taskID string, confirmOverwrite bool) error {
	task, err := e.backupUC.GetMigrationTaskEntity(ctx, taskID)
	if err != nil {
		return domainBackup.ErrMigrationTaskNotFound
	}
	if task.Mode == domainBackup.MigrationModeOverwrite && !confirmOverwrite {
		return domainBackup.ErrOverwriteConfirmRequired
	}

	go e.executeTask(task)
	return nil
}

func (e *MigrationExecutor) executeTask(task *domainBackup.MigrationTask) {
	ctx := context.Background()
	startedAt := time.Now()
	record := &domainBackup.MigrationRecord{
		TaskID:     task.ID,
		TaskName:   task.Name,
		DBType:     task.DBType,
		Mode:       task.Mode,
		Status:     domainBackup.MigrationStatusRunning,
		SourceHost: task.SourceHost,
		TargetHost: task.TargetHost,
		DBNames:    task.DBNames,
		StartedAt:  startedAt,
	}
	if err := e.backupUC.CreateMigrationRecord(ctx, record); err != nil {
		log.Printf("[MigrationExecutor] create record failed: %v", err)
		return
	}

	dbNames := parseDBNames(task.DBNames)
	if len(dbNames) == 0 {
		e.finishRecord(ctx, task, record, 0, 0, 1, "数据库列表为空")
		return
	}

	sourcePassword, err := e.backupUC.DecryptMigrationPassword(task.SourcePassword)
	if err != nil {
		e.finishRecord(ctx, task, record, 0, 0, len(dbNames), fmt.Sprintf("解密源数据库密码失败: %v", err))
		return
	}
	targetPassword, err := e.backupUC.DecryptMigrationPassword(task.TargetPassword)
	if err != nil {
		e.finishRecord(ctx, task, record, 0, 0, len(dbNames), fmt.Sprintf("解密目标数据库密码失败: %v", err))
		return
	}

	var successCount, skippedCount, failedCount int
	for _, dbName := range dbNames {
		item := e.executeDB(ctx, task, record.ID, dbName, sourcePassword, targetPassword)
		if err := e.backupUC.CreateMigrationRecordItem(ctx, item); err != nil {
			log.Printf("[MigrationExecutor] create item failed: %v", err)
		}
		switch item.Status {
		case domainBackup.MigrationStatusSuccess:
			successCount++
		case domainBackup.MigrationStatusSkipped:
			skippedCount++
		default:
			failedCount++
		}
	}

	e.finishRecord(ctx, task, record, successCount, skippedCount, failedCount, "")
}

func (e *MigrationExecutor) executeDB(ctx context.Context, task *domainBackup.MigrationTask, recordID, dbName, sourcePassword, targetPassword string) *domainBackup.MigrationRecordItem {
	startedAt := time.Now()
	item := &domainBackup.MigrationRecordItem{
		RecordID:  recordID,
		DBName:    dbName,
		Status:    domainBackup.MigrationStatusRunning,
		StartedAt: startedAt,
	}
	defer func() {
		now := time.Now()
		item.FinishedAt = &now
		item.Duration = int64(now.Sub(startedAt).Seconds())
	}()

	if sameDatabaseEndpoint(task, dbName) {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = "源端和目标端相同，禁止覆盖同名数据库"
		return item
	}

	sourceExists, err := e.databaseExists(task.DBType, task.SourceHost, task.SourcePort, task.SourceUser, sourcePassword, dbName)
	if err != nil {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = fmt.Sprintf("检查源库失败: %v", err)
		return item
	}
	if !sourceExists {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = "源库不存在"
		return item
	}

	targetExists, err := e.databaseExists(task.DBType, task.TargetHost, task.TargetPort, task.TargetUser, targetPassword, dbName)
	if err != nil {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = fmt.Sprintf("检查目标库失败: %v", err)
		return item
	}
	if task.Mode == domainBackup.MigrationModeCreateIfMissing && targetExists {
		item.Action = domainBackup.MigrationActionSkipped
		item.Status = domainBackup.MigrationStatusSkipped
		item.Message = "目标库已存在，已跳过"
		return item
	}

	if task.Mode == domainBackup.MigrationModeOverwrite && targetExists {
		if err := e.dropDatabase(task.DBType, task.TargetHost, task.TargetPort, task.TargetUser, targetPassword, dbName); err != nil {
			item.Status = domainBackup.MigrationStatusFailed
			item.Message = fmt.Sprintf("删除目标库失败: %v", err)
			return item
		}
		item.Action = domainBackup.MigrationActionOverwritten
	} else {
		item.Action = domainBackup.MigrationActionCreated
	}

	if err := e.createDatabase(task.DBType, task.TargetHost, task.TargetPort, task.TargetUser, targetPassword, dbName); err != nil {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = fmt.Sprintf("创建目标库失败: %v", err)
		return item
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("migration_%s_%s%s", safeFilePart(dbName), time.Now().Format("20060102_150405"), migrationDumpExt(task.DBType)))
	defer os.Remove(tmpFile)

	if err := e.dumpDatabase(task.DBType, task.SourceHost, task.SourcePort, task.SourceUser, sourcePassword, dbName, tmpFile); err != nil {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = fmt.Sprintf("导出源库失败: %v", err)
		return item
	}
	if err := e.restoreDatabase(task.DBType, task.TargetHost, task.TargetPort, task.TargetUser, targetPassword, dbName, tmpFile); err != nil {
		item.Status = domainBackup.MigrationStatusFailed
		item.Message = fmt.Sprintf("导入目标库失败: %v", err)
		return item
	}

	item.Status = domainBackup.MigrationStatusSuccess
	item.Message = "迁移成功"
	return item
}

func (e *MigrationExecutor) finishRecord(ctx context.Context, task *domainBackup.MigrationTask, record *domainBackup.MigrationRecord, successCount, skippedCount, failedCount int, errMessage string) {
	now := time.Now()
	record.FinishedAt = &now
	record.Duration = int64(now.Sub(record.StartedAt).Seconds())
	record.Summary = fmt.Sprintf("成功 %d 个，跳过 %d 个，失败 %d 个", successCount, skippedCount, failedCount)
	record.Error = errMessage

	switch {
	case failedCount == 0:
		record.Status = domainBackup.MigrationStatusSuccess
	case successCount > 0 || skippedCount > 0:
		record.Status = domainBackup.MigrationStatusPartialSuccess
	default:
		record.Status = domainBackup.MigrationStatusFailed
	}
	if err := e.backupUC.UpdateMigrationRecord(ctx, record); err != nil {
		log.Printf("[MigrationExecutor] update record failed: %v", err)
	}
	if err := e.backupUC.UpdateMigrationTaskLastRun(ctx, task.ID, record.Status); err != nil {
		log.Printf("[MigrationExecutor] update task last run failed: %v", err)
	}
}

func (e *MigrationExecutor) databaseExists(dbType domainBackup.BackupType, host string, port int, user, password, dbName string) (bool, error) {
	var query string
	switch dbType {
	case domainBackup.BackupTypePostgres:
		query = fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname=%s", sqlString(dbName))
		cmd := fmt.Sprintf("PGPASSWORD=%s psql -h %s -p %s -U %s -d postgres -tAc %s",
			shellEscape(password), shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), shellEscape(query))
		out, err := runShell(cmd)
		return strings.TrimSpace(out) == "1", err
	case domainBackup.BackupTypeMySQL:
		query = fmt.Sprintf("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME=%s", sqlString(dbName))
		cmd := fmt.Sprintf("mysql -h %s -P %s -u %s %s -N -e %s",
			shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), mysqlPasswordArg(password), shellEscape(query))
		out, err := runShell(cmd)
		return strings.TrimSpace(out) == dbName, err
	default:
		return false, fmt.Errorf("不支持的迁移数据库类型: %s", dbType)
	}
}

func (e *MigrationExecutor) createDatabase(dbType domainBackup.BackupType, host string, port int, user, password, dbName string) error {
	switch dbType {
	case domainBackup.BackupTypePostgres:
		cmd := fmt.Sprintf("PGPASSWORD=%s createdb -h %s -p %s -U %s %s",
			shellEscape(password), shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), shellEscape(dbName))
		_, err := runShell(cmd)
		return err
	case domainBackup.BackupTypeMySQL:
		query := fmt.Sprintf("CREATE DATABASE %s", mysqlIdentifier(dbName))
		cmd := fmt.Sprintf("mysql -h %s -P %s -u %s %s -e %s",
			shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), mysqlPasswordArg(password), shellEscape(query))
		_, err := runShell(cmd)
		return err
	default:
		return fmt.Errorf("不支持的迁移数据库类型: %s", dbType)
	}
}

func (e *MigrationExecutor) dropDatabase(dbType domainBackup.BackupType, host string, port int, user, password, dbName string) error {
	switch dbType {
	case domainBackup.BackupTypePostgres:
		terminate := fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=%s", sqlString(dbName))
		terminateCmd := fmt.Sprintf("PGPASSWORD=%s psql -h %s -p %s -U %s -d postgres -tAc %s",
			shellEscape(password), shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), shellEscape(terminate))
		if _, err := runShell(terminateCmd); err != nil {
			return err
		}
		cmd := fmt.Sprintf("PGPASSWORD=%s dropdb -h %s -p %s -U %s --if-exists %s",
			shellEscape(password), shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), shellEscape(dbName))
		_, err := runShell(cmd)
		return err
	case domainBackup.BackupTypeMySQL:
		query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", mysqlIdentifier(dbName))
		cmd := fmt.Sprintf("mysql -h %s -P %s -u %s %s -e %s",
			shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), mysqlPasswordArg(password), shellEscape(query))
		_, err := runShell(cmd)
		return err
	default:
		return fmt.Errorf("不支持的迁移数据库类型: %s", dbType)
	}
}

func (e *MigrationExecutor) dumpDatabase(dbType domainBackup.BackupType, host string, port int, user, password, dbName, outputFile string) error {
	switch dbType {
	case domainBackup.BackupTypePostgres:
		cmd := fmt.Sprintf("PGPASSWORD=%s pg_dump -h %s -p %s -U %s -d %s -Fc -f %s",
			shellEscape(password), shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), shellEscape(dbName), shellEscape(outputFile))
		_, err := runShell(cmd)
		return err
	case domainBackup.BackupTypeMySQL:
		cmd := fmt.Sprintf("mysqldump -h %s -P %s -u %s %s %s > %s",
			shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), mysqlPasswordArg(password), shellEscape(dbName), shellEscape(outputFile))
		_, err := runShell(cmd)
		return err
	default:
		return fmt.Errorf("不支持的迁移数据库类型: %s", dbType)
	}
}

func (e *MigrationExecutor) restoreDatabase(dbType domainBackup.BackupType, host string, port int, user, password, dbName, inputFile string) error {
	switch dbType {
	case domainBackup.BackupTypePostgres:
		cmd := fmt.Sprintf("PGPASSWORD=%s pg_restore -h %s -p %s -U %s -d %s %s",
			shellEscape(password), shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), shellEscape(dbName), shellEscape(inputFile))
		_, err := runShell(cmd)
		return err
	case domainBackup.BackupTypeMySQL:
		cmd := fmt.Sprintf("mysql -h %s -P %s -u %s %s %s < %s",
			shellEscape(host), shellEscape(fmt.Sprintf("%d", port)), shellEscape(user), mysqlPasswordArg(password), shellEscape(dbName), shellEscape(inputFile))
		_, err := runShell(cmd)
		return err
	default:
		return fmt.Errorf("不支持的迁移数据库类型: %s", dbType)
	}
}

func runShell(shellCmd string) (string, error) {
	cmd := exec.Command("sh", "-c", shellCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%w: %s", err, string(output))
	}
	return string(output), nil
}

func sameDatabaseEndpoint(task *domainBackup.MigrationTask, dbName string) bool {
	return strings.EqualFold(task.SourceHost, task.TargetHost) &&
		task.SourcePort == task.TargetPort &&
		strings.TrimSpace(dbName) != ""
}

func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func mysqlIdentifier(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func mysqlPasswordArg(password string) string {
	return shellEscape("-p" + password)
}

func migrationDumpExt(dbType domainBackup.BackupType) string {
	if dbType == domainBackup.BackupTypePostgres {
		return ".dump"
	}
	return ".sql"
}
