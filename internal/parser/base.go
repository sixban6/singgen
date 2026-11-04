package parser

import (
	"log/slog"

	"github.com/sixban6/singgen/internal/logx"
)

// base 为各协议解析器提供可注入的 logger。
type base struct {
	log *slog.Logger
}

// SetLogger 设置解析器使用的 logger；为 nil 时不输出日志。
func (b *base) SetLogger(l *slog.Logger) { b.log = l }

func (b *base) logger() *slog.Logger { return logx.Or(b.log) }

// loggerSetter 由内嵌 base 的解析器实现
type loggerSetter interface {
	SetLogger(*slog.Logger)
}

// WithLogger 为解析器注入 logger（解析器不支持时原样返回）。
func WithLogger(p Parser, l *slog.Logger) Parser {
	if ls, ok := p.(loggerSetter); ok {
		ls.SetLogger(l)
	}
	return p
}
