package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"ops-hub/internal/application/backup"
	appHost "ops-hub/internal/application/host"
	domainBackup "ops-hub/internal/domain/backup"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/ssh"
)

// Scheduler 备份调度器
type Scheduler struct {
	backupUC *backup.UseCase
	hostUC   *appHost.UseCase
	cron     *cron.Cron
	entryMap map[string]cron.EntryID // taskID -> entryID
	stopCh   chan struct{}
}

// NewScheduler 创建备份调度器
func NewScheduler(backupUC *backup.UseCase, hostUC *appHost.UseCase) *Scheduler {
	return &Scheduler{
		backupUC: backupUC,
		hostUC:   hostUC,
		cron:     cron.New(cron.WithSeconds()),
		entryMap: make(map[string]cron.EntryID),
		stopCh:   make(chan struct{}),
	}
}

// Start 启动调度器，加载所有启用的任务
func (s *Scheduler) Start() error {
	ctx := context.Background()
	tasks, err := s.backupUC.GetAllEnabledTasks(ctx)
	if err != nil {
		return fmt.Errorf("加载备份任务失败: %w", err)
	}

	for _, task := range tasks {
		if err := s.addTask(task); err != nil {
			log.Printf("[BackupScheduler] 添加任务失败 task=%s err=%v", task.Name, err)
		}
	}

	s.cron.Start()
	log.Printf("[BackupScheduler] started, loaded %d tasks", len(tasks))
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	close(s.stopCh)
}

// Reload 重新加载所有任务（任务增删改后调用）
func (s *Scheduler) Reload() {
	// 移除所有现有任务
	for taskID, entryID := range s.entryMap {
		s.cron.Remove(entryID)
		delete(s.entryMap, taskID)
	}

	ctx := context.Background()
	tasks, err := s.backupUC.GetAllEnabledTasks(ctx)
	if err != nil {
		log.Printf("[BackupScheduler] reload failed: %v", err)
		return
	}

	for _, task := range tasks {
		if err := s.addTask(task); err != nil {
			log.Printf("[BackupScheduler] reload add task=%s err=%v", task.Name, err)
		}
	}
	log.Printf("[BackupScheduler] reloaded, %d tasks active", len(tasks))
}

// TriggerNow 手动触发一个任务（允许对禁用任务执行）
func (s *Scheduler) TriggerNow(ctx context.Context, taskID string) error {
	task, err := s.backupUC.GetTaskEntity(ctx, taskID)
	if err != nil {
		return domainBackup.ErrTaskNotFound
	}

	go s.executeTask(task)
	return nil
}

func (s *Scheduler) addTask(task *domainBackup.BackupTask) error {
	t := task // capture
	// robfig/cron v3 with seconds: prepend "0" to standard 5-field cron to make 6-field
	cronExpr := "0 " + t.CronExpr // e.g. "0 0 2 * * *" for daily at 02:00
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeTask(t)
	})
	if err != nil {
		return fmt.Errorf("cron 表达式无效: %w", err)
	}
	s.entryMap[t.ID] = entryID
	return nil
}

// executeTask 执行一个备份任务完整流程
func (s *Scheduler) executeTask(task *domainBackup.BackupTask) {
	ctx := context.Background()
	startedAt := time.Now()

	// 创建记录
	record := &domainBackup.BackupRecord{
		TaskID:    task.ID,
		TaskName:  task.Name,
		Status:    domainBackup.BackupStatusRunning,
		StartedAt: startedAt,
	}
	if err := s.backupUC.CreateRecord(ctx, record); err != nil {
		log.Printf("[BackupScheduler] create record failed: %v", err)
		return
	}

	// 执行备份
	fileName, fileSize, err := s.doBackup(ctx, task)

	// 更新记录
	now := time.Now()
	record.FinishedAt = &now
	record.Duration = int64(now.Sub(startedAt).Seconds())

	if err != nil {
		record.Status = domainBackup.BackupStatusFailed
		record.Error = err.Error()
		log.Printf("[BackupScheduler] task=%s FAILED: %v", task.Name, err)
	} else {
		record.Status = domainBackup.BackupStatusSuccess
		record.FileName = fileName
		record.FileSize = fileSize
		log.Printf("[BackupScheduler] task=%s SUCCESS file=%s size=%d", task.Name, fileName, fileSize)
	}

	_ = s.backupUC.UpdateRecord(ctx, record)
	_ = s.backupUC.UpdateTaskLastRun(ctx, task.ID, record.Status)

	// 清理过期备份
	if err == nil && task.RetentionDays > 0 {
		s.cleanOldBackups(ctx, task)
	}
}

// doBackup 执行备份核心逻辑：本地 dump → SFTP 推送 → 清理临时文件
func (s *Scheduler) doBackup(ctx context.Context, task *domainBackup.BackupTask) (string, int64, error) {
	// 1. 解密数据库密码
	dbPassword, err := s.backupUC.DecryptDBPassword(task.DBPassword)
	if err != nil {
		return "", 0, fmt.Errorf("解密数据库密码失败: %w", err)
	}

	// 2. 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	dbNames := parseDBNames(task.DBName)
	fileName := backupFileName(task.BackupType, dbNames, timestamp)
	localTmpDir, localTmpFile, err := newBackupTempFile(fileName)
	if err != nil {
		return "", 0, err
	}
	defer os.RemoveAll(localTmpDir)

	// 3. 本地执行 dump
	if err := s.localDump(task, dbPassword, localTmpFile); err != nil {
		return "", 0, fmt.Errorf("数据库导出失败: %w", err)
	}

	// 4. 获取本地文件大小
	stat, err := os.Stat(localTmpFile)
	if err != nil {
		return "", 0, fmt.Errorf("获取备份文件信息失败: %w", err)
	}
	fileSize := stat.Size()

	// 5. SFTP 传输到目标宿主机
	if err := s.transferViaSFTP(ctx, task.TargetHostID, localTmpFile, task.TargetPath, fileName); err != nil {
		return "", 0, fmt.Errorf("SFTP 传输失败: %w", err)
	}

	return fileName, fileSize, nil
}

func newBackupTempFile(fileName string) (string, string, error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "backup_run_")
	if err != nil {
		return "", "", fmt.Errorf("创建备份临时目录失败: %w", err)
	}
	return tmpDir, filepath.Join(tmpDir, fileName), nil
}

// localDump 在本地执行数据库导出
func (s *Scheduler) localDump(task *domainBackup.BackupTask, dbPassword, outputFile string) error {
	var cmd *exec.Cmd
	dbNames := parseDBNames(task.DBName)

	switch task.BackupType {
	case domainBackup.BackupTypePostgres:
		baseArgs := []string{
			"-h", task.DBHost,
			"-p", fmt.Sprintf("%d", task.DBPort),
			"-U", task.DBUser,
		}
		if len(dbNames) == 0 {
			allDBNames, err := listPostgresDatabases(dbPassword, baseArgs)
			if err != nil {
				return fmt.Errorf("查询 PostgreSQL 数据库列表失败: %w", err)
			}
			if len(allDBNames) == 0 {
				return fmt.Errorf("未查询到可备份的 PostgreSQL 数据库")
			}
			if err := dumpPostgresDatabases(dbPassword, baseArgs, allDBNames, outputFile); err != nil {
				return err
			}
			return nil
		} else if len(dbNames) == 1 {
			args := append(append([]string{}, baseArgs...), "-d", dbNames[0], "-Fc", "-f", outputFile)
			shellCmd := fmt.Sprintf("PGPASSWORD=%s pg_dump %s", shellEscape(dbPassword), shellJoin(args))
			cmd = exec.Command("sh", "-c", shellCmd)
		} else {
			if err := dumpPostgresDatabases(dbPassword, baseArgs, dbNames, outputFile); err != nil {
				return err
			}
			return nil
		}

	case domainBackup.BackupTypeMySQL:
		args := []string{
			fmt.Sprintf("-h%s", task.DBHost),
			fmt.Sprintf("-P%d", task.DBPort),
			fmt.Sprintf("-u%s", task.DBUser),
			fmt.Sprintf("-p%s", dbPassword),
		}
		if len(dbNames) == 0 {
			args = append(args, "--all-databases")
		} else {
			args = append(args, "--databases")
			args = append(args, dbNames...)
		}
		shellCmd := fmt.Sprintf("mysqldump %s | gzip > %s", shellJoin(args), shellEscape(outputFile))
		cmd = exec.Command("sh", "-c", shellCmd)

	default:
		return fmt.Errorf("不支持的备份类型: %s", task.BackupType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// transferViaSFTP 通过 SFTP 传输文件到目标宿主机
func (s *Scheduler) transferViaSFTP(ctx context.Context, hostID, localFile, remotePath, remoteFileName string) error {
	// 从宿主机连接池获取 SSH 连接
	sshClient, err := s.hostUC.GetSSH(ctx, hostID)
	if err != nil {
		return fmt.Errorf("获取 SSH 连接失败: %w", err)
	}

	// 创建 SFTP 客户端（基于共享 SSH 连接）
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("创建 SFTP 客户端失败: %w", err)
	}
	defer sftpClient.Close()

	// 确保远程目录存在
	if err := sftpClient.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("创建远程目录失败: %w", err)
	}

	// 打开本地文件
	localF, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer localF.Close()

	// 创建远程文件
	remoteFilePath := remotePath + "/" + remoteFileName
	remoteF, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remoteF.Close()

	// 拷贝数据
	if _, err := io.Copy(remoteF, localF); err != nil {
		return fmt.Errorf("文件传输失败: %w", err)
	}

	return nil
}

// cleanOldBackups 通过 SSH 清理目标宿主机上超过保留天数的备份文件
func (s *Scheduler) cleanOldBackups(ctx context.Context, task *domainBackup.BackupTask) {
	sshClient, err := s.hostUC.GetSSH(ctx, task.TargetHostID)
	if err != nil {
		log.Printf("[BackupScheduler] clean: SSH connect failed for host=%s: %v", task.TargetHostID, err)
		return
	}

	session, err := sshClient.NewSession()
	if err != nil {
		log.Printf("[BackupScheduler] clean: create session failed: %v", err)
		return
	}
	defer session.Close()

	// find 删除超过保留天数的备份文件
	cmd := fmt.Sprintf("find %s \\( -name '*.sql.gz' -o -name '*.dump' -o -name '*.tar.gz' \\) -mtime +%d -delete",
		shellEscape(task.TargetPath), task.RetentionDays)
	if out, runErr := session.CombinedOutput(cmd); runErr != nil {
		log.Printf("[BackupScheduler] clean: cmd failed: %v output=%s", runErr, string(out))
	}
}

type archiveFile struct {
	Path string
	Name string
}

func backupFileName(backupType domainBackup.BackupType, dbNames []string, timestamp string) string {
	scope := "selected"
	if len(dbNames) == 0 {
		scope = "all"
	} else if len(dbNames) == 1 {
		scope = safeFilePart(dbNames[0])
	}
	if backupType == domainBackup.BackupTypePostgres {
		if len(dbNames) == 1 {
			return fmt.Sprintf("%s_%s_%s.dump", string(backupType), scope, timestamp)
		}
		return fmt.Sprintf("%s_%s_%s.tar.gz", string(backupType), scope, timestamp)
	}
	return fmt.Sprintf("%s_%s_%s.sql.gz", string(backupType), scope, timestamp)
}

func listPostgresDatabases(dbPassword string, baseArgs []string) ([]string, error) {
	query := "SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate ORDER BY datname"
	args := append(append([]string{}, baseArgs...), "-d", "postgres", "-tAc", query)
	shellCmd := fmt.Sprintf("PGPASSWORD=%s psql %s", shellEscape(dbPassword), shellJoin(args))
	output, err := exec.Command("sh", "-c", shellCmd).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(output))
	}

	lines := strings.FieldsFunc(string(output), func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	dbNames := make([]string, 0, len(lines))
	for _, line := range lines {
		dbName := strings.TrimSpace(line)
		if dbName != "" {
			dbNames = append(dbNames, dbName)
		}
	}
	return dbNames, nil
}

func dumpPostgresDatabases(dbPassword string, baseArgs []string, dbNames []string, outputFile string) error {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "backup_postgres_")
	if err != nil {
		return fmt.Errorf("创建 PostgreSQL 备份临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archiveRoot := strings.TrimSuffix(filepath.Base(outputFile), ".tar.gz")
	files := make([]archiveFile, 0, len(dbNames))
	for _, dbName := range dbNames {
		dumpFileName := fmt.Sprintf("%s.dump", safeFilePart(dbName))
		dumpPath := filepath.Join(tmpDir, dumpFileName)
		args := append(append([]string{}, baseArgs...), "-d", dbName, "-Fc", "-f", dumpPath)
		shellCmd := fmt.Sprintf("PGPASSWORD=%s pg_dump %s", shellEscape(dbPassword), shellJoin(args))
		output, err := exec.Command("sh", "-c", shellCmd).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, string(output))
		}
		files = append(files, archiveFile{
			Path: dumpPath,
			Name: filepath.ToSlash(filepath.Join(archiveRoot, dumpFileName)),
		})
	}
	if err := writeTarGz(outputFile, files); err != nil {
		return fmt.Errorf("打包 PostgreSQL 数据库备份失败: %w", err)
	}
	return nil
}

func writeTarGz(outputFile string, files []archiveFile) error {
	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, file := range files {
		info, err := os.Stat(file.Path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = file.Name
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		in, err := os.Open(file.Path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, in); err != nil {
			_ = in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
	}
	return nil
}

// shellEscape 安全的 shell 转义（单引号包裹）
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func shellJoin(args []string) string {
	escaped := make([]string, len(args))
	for i, arg := range args {
		escaped[i] = shellEscape(arg)
	}
	return strings.Join(escaped, " ")
}

func parseDBNames(dbName string) []string {
	parts := strings.FieldsFunc(dbName, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	names := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func safeFilePart(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	return replacer.Replace(s)
}

// SSHProvider 接口适配（hostUC 已实现）
var _ sshProvider = (*appHost.UseCase)(nil)

type sshProvider interface {
	GetSSH(ctx context.Context, hostID string) (*ssh.Client, error)
}
