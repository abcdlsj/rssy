package internal

import (
	"context"
	"time"

	"github.com/abcdlsj/cr"
	"github.com/charmbracelet/log"
	"gorm.io/gorm/logger"
)

type Logger struct {
	log    *log.Logger
	prefix string
}

func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *Logger) Info(ctx context.Context, format string, v ...interface{}) {
	l.log.Infof(l.prefix+format, v...)
}

func (l *Logger) Warn(ctx context.Context, format string, v ...interface{}) {
	l.log.Warnf(l.prefix+format, v...)
}

func (l *Logger) Error(ctx context.Context, format string, v ...interface{}) {
	l.log.Errorf(l.prefix+format, v...)
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	l.log.Infof(l.prefix+"%s|rows:%d|error:%v|time:%s", cr.PLCyan(sql), rows, err, time.Since(begin))
}
