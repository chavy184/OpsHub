package config

import (
	"fmt"
	"strings"

	"ops-hub/pkg/logger"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Security SecurityConfig `mapstructure:"security"`
	Static   StaticConfig   `mapstructure:"static"`
	Jenkins  JenkinsConfig  `mapstructure:"jenkins"`
	Log      logger.Config  `mapstructure:"log"`
	Document DocumentConfig `mapstructure:"document"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	DSN     string `mapstructure:"dsn"`
	LogFile string `mapstructure:"log_file"`
}

type SecurityConfig struct {
	EncryptKey    string `mapstructure:"encrypt_key"`
	AuthSecret    string `mapstructure:"auth_secret"`
	AdminUsername string `mapstructure:"admin_username"`
	AdminPassword string `mapstructure:"admin_password"`
}

type StaticConfig struct {
	Dir string `mapstructure:"dir"`
}

type JenkinsConfig struct {
	URL   string `mapstructure:"url"`
	User  string `mapstructure:"user"`
	Token string `mapstructure:"token"`
}

type DocumentConfig struct {
	BaseDir     string `mapstructure:"base_dir"`
	MaxFileSize int64  `mapstructure:"max_file_size"` // 单文件最大字节数
}

// Load 从配置文件加载，未配置的项使用默认值
func Load(path string) (*Config, error) {
	v := viper.New()

	// 默认值
	v.SetDefault("server.port", 48884)
	v.SetDefault("database.dsn", "")
	v.SetDefault("database.log_file", "logs/gorm.log")
	v.SetDefault("security.encrypt_key", "")
	v.SetDefault("security.auth_secret", "")
	v.SetDefault("security.admin_username", "admin")
	v.SetDefault("security.admin_password", "")
	v.SetDefault("static.dir", "./static")
	v.SetDefault("jenkins.url", "")
	v.SetDefault("jenkins.user", "")
	v.SetDefault("jenkins.token", "")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.file_path", "logs/ops-hub.log")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.max_age", 30)
	v.SetDefault("log.compress", true)
	v.SetDefault("document.base_dir", "./documents")
	v.SetDefault("document.max_file_size", 10485760)

	// OPSHUB_DATABASE_DSN、OPSHUB_SECURITY_ENCRYPT_KEY 等环境变量可覆盖配置文件。
	v.SetEnvPrefix("OPSHUB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// 保留现有部署中已使用的简写环境变量。
	_ = v.BindEnv("security.encrypt_key", "OPSHUB_ENCRYPT_KEY", "OPSHUB_SECURITY_ENCRYPT_KEY")
	_ = v.BindEnv("security.auth_secret", "OPSHUB_AUTH_SECRET", "OPSHUB_SECURITY_AUTH_SECRET")
	_ = v.BindEnv("security.admin_username", "OPSHUB_ADMIN_USERNAME", "OPSHUB_SECURITY_ADMIN_USERNAME")
	_ = v.BindEnv("security.admin_password", "OPSHUB_ADMIN_PASSWORD", "OPSHUB_SECURITY_ADMIN_PASSWORD")

	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if !strings.Contains(err.Error(), "no such file") &&
				!strings.Contains(err.Error(), "cannot find") {
				return nil, err
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return fmt.Errorf("database.dsn 未配置：请设置 OPSHUB_DATABASE_DSN 或使用本地配置文件")
	}
	if len(cfg.Security.EncryptKey) != 32 {
		return fmt.Errorf("security.encrypt_key 必须为 32 字节：请设置 OPSHUB_ENCRYPT_KEY")
	}
	if len(cfg.Security.AuthSecret) < 32 {
		return fmt.Errorf("security.auth_secret 至少需要 32 字节：请设置 OPSHUB_AUTH_SECRET")
	}
	if strings.TrimSpace(cfg.Security.AdminUsername) == "" {
		return fmt.Errorf("security.admin_username 不能为空")
	}
	if len(cfg.Security.AdminPassword) < 12 {
		return fmt.Errorf("security.admin_password 至少需要 12 个字符：请设置 OPSHUB_ADMIN_PASSWORD")
	}
	return nil
}
