package server

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

const fpLogPath = "/data/logs/fp.log"

var (
	fpLogOnce sync.Once
	fpLog     *lumberjack.Logger
)

func getFPLog() *lumberjack.Logger {
	fpLogOnce.Do(func() {
		_ = os.MkdirAll(filepath.Dir(fpLogPath), 0755)
		fpLog = &lumberjack.Logger{
			Filename:   fpLogPath,
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     3,
			Compress:   true,
		}
	})
	return fpLog
}
