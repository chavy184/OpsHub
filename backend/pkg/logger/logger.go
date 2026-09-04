package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config 日志配置
type Config struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	FilePath   string `mapstructure:"file_path"`   // 日志文件路径，空则仅输出到 stdout
	MaxSize    int    `mapstructure:"max_size"`    // 单文件最大 MB
	MaxBackups int    `mapstructure:"max_backups"` // 保留旧文件数量
	MaxAge     int    `mapstructure:"max_age"`     // 保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧文件
}

// DefaultConfig 返回默认日志配置
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		FilePath:   "logs/ops-hub.log",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
}

// Init 初始化全局 zap logger
func Init(cfg Config) *zap.Logger {
	level := parseLevel(cfg.Level)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	// 控制台输出（JSON 格式，便于 docker logs 解析）
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), level)
	cores = append(cores, consoleCore)

	// 文件输出（JSON 格式 + 轮转）
	if cfg.FilePath != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(fileWriter), level)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	l := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// 替换全局 logger
	zap.ReplaceGlobals(l)

	return l
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
