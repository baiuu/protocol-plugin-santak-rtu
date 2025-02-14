package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"tp-santak-rtu/internal/config"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ANSI颜色码
const (
	colorRed    = 31
	colorGreen  = 32
	colorYellow = 33
	colorBlue   = 36
	colorGray   = 37
)

type CustomFormatter struct {
	logrus.TextFormatter
	isTerminal bool
}

func getColorByLevel(level logrus.Level) int {
	switch level {
	case logrus.ErrorLevel:
		return colorRed
	case logrus.WarnLevel:
		return colorYellow
	case logrus.InfoLevel:
		return colorGreen
	case logrus.DebugLevel:
		return colorBlue
	default:
		return colorGray
	}
}

// 添加颜色包装
func colored(color int, text string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, text)
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// 处理文件路径
	var filePath string
	if entry.Caller != nil {
		file := entry.Caller.File
		if idx := strings.Index(file, "internal"); idx != -1 {
			filePath = file[idx:]
		} else {
			filePath = entry.Caller.File
		}
	}

	// 获取level对应的颜色
	levelColor := getColorByLevel(entry.Level)

	// 构建日志级别部分（带颜色）
	levelText := strings.ToUpper(entry.Level.String())
	if f.isTerminal {
		levelText = colored(levelColor, levelText)
	}

	// 构建时间戳部分（使用灰色）
	timeText := timestamp
	if f.isTerminal {
		timeText = colored(colorGray, timestamp)
	}

	// 构建文件路径部分（使用蓝色）
	fileInfo := fmt.Sprintf("%s:%d", filePath, entry.Caller.Line)
	if f.isTerminal {
		fileInfo = colored(colorBlue, fileInfo)
	}

	// 构建字段信息
	var fields string
	if len(entry.Data) > 0 {
		parts := make([]string, 0, len(entry.Data))
		for k, v := range entry.Data {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		fields = strings.Join(parts, " ")
	}

	// 构建完整的日志消息
	var logMessage string
	if len(fields) > 0 {
		logMessage = fmt.Sprintf("%s[%s]%s | %s | %s",
			levelText, // 带颜色的日志级别
			timeText,  // 带颜色的时间戳
			fileInfo,  // 带颜色的文件信息
			//entry.Caller.Function, // 函数名
			fields,        // 字段信息
			entry.Message, // 原始消息
		)
	} else {
		logMessage = fmt.Sprintf("%s[%s]%s %s",
			levelText, // 带颜色的日志级别
			timeText,  // 带颜色的时间戳
			fileInfo,  // 带颜色的文件信息
			//entry.Caller.Function, // 函数名
			entry.Message, // 原始消息
		)
	}

	return []byte(logMessage + "\n"), nil
}

// InitLogger 初始化日志系统
func InitLogger(cfg *config.LogConfig) {
	// 1. 创建文件日志写入器
	fileLogger := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 2. 创建多重输出
	multiWriter := io.MultiWriter(os.Stdout, fileLogger)

	// 3. 设置日志输出
	logrus.SetOutput(multiWriter)

	// 4. 启用调用者信息报告
	logrus.SetReportCaller(true)

	// 5. 设置自定义格式化器
	logrus.SetFormatter(&CustomFormatter{
		isTerminal: true, // 启用终端颜色支持
	})

	// 6. 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
		logrus.Warnf("无效的日志级别配置: %s, 使用默认级别: INFO", cfg.Level)
	}
	logrus.SetLevel(level)
}
