package server

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const fpLogPath = "/data/logs/fp.log"

var (
	fpLogMu sync.Mutex
	fpLog   *lumberjack.Logger
)

func newFPLogger() *lumberjack.Logger {
	_ = os.MkdirAll(filepath.Dir(fpLogPath), 0755)
	return &lumberjack.Logger{
		Filename:   fpLogPath,
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     3,
		Compress:   true,
	}
}

func init() {
	// 外部对 fp.log 做 mv/truncate/logrotate 后，进程可能仍持有已 unlink 的 inode，
	// 写入不报错但磁盘上当前路径不再增长；定期关句柄可重新打开真实路径。
	go func() {
		t := time.NewTicker(24 * time.Hour)
		for range t.C {
			fpLogMu.Lock()
			if fpLog != nil {
				_ = fpLog.Close()
				fpLog = nil
			}
			fpLogMu.Unlock()
		}
	}()
}

// writeFPLogLine 写入一行 JSON；失败时关闭并换新 Logger 重试一次。
func writeFPLogLine(b []byte) error {
	fpLogMu.Lock()
	defer fpLogMu.Unlock()

	doWrite := func() error {
		if fpLog == nil {
			fpLog = newFPLogger()
		}
		_, err := fpLog.Write(b)
		return err
	}

	if err := doWrite(); err != nil {
		if fpLog != nil {
			_ = fpLog.Close()
		}
		fpLog = newFPLogger()
		_, err2 := fpLog.Write(b)
		return err2
	}
	return nil
}
