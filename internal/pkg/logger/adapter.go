// internal/pkg/logger/adapter.go
package logger

import (
	"log"

	"github.com/sirupsen/logrus"
)

// LogrusAdapter 将 logrus.Logger 适配为标准 log.Logger 接口
type LogrusAdapter struct {
	logger *logrus.Logger
	prefix string
}

// NewLogrusAdapter 创建一个新的日志适配器
func NewLogrusAdapter(logger *logrus.Logger, prefix string) *log.Logger {
	return log.New(&logrusWriter{logger: logger}, prefix, 0)
}

// logrusWriter 实现 io.Writer 接口，将标准库日志输出转发到 logrus
type logrusWriter struct {
	logger *logrus.Logger
}

// Write 实现 io.Writer 接口，将日志消息转发到 logrus
func (w *logrusWriter) Write(p []byte) (int, error) {
	// 移除标准库logger添加的额外换行符
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	// 使用logrus记录日志
	w.logger.Info(msg)
	return len(p), nil
}

// AdapterOption 定义适配器选项函数类型
type AdapterOption func(*LogrusAdapter)

// WithPrefix 设置日志前缀
func WithPrefix(prefix string) AdapterOption {
	return func(a *LogrusAdapter) {
		a.prefix = prefix
	}
}

// CreateAdapter 创建一个配置完善的日志适配器
func CreateAdapter(logger *logrus.Logger, opts ...AdapterOption) *log.Logger {
	adapter := &LogrusAdapter{
		logger: logger,
		prefix: "[SANTAK-RTU] ", // 默认前缀
	}

	// 应用选项
	for _, opt := range opts {
		opt(adapter)
	}

	return NewLogrusAdapter(logger, adapter.prefix)
}
