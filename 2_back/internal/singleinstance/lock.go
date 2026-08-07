package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const lockName = "rkw_hatcher.lock"

// Acquire 保证同时只有一个实例：若已有进程则强制结束再占用锁。
func Acquire(baseDir string) (release func(), err error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(baseDir, lockName)

	if raw, e := os.ReadFile(lockPath); e == nil {
		pidStr := strings.TrimSpace(string(raw))
		if oldPID, e2 := strconv.Atoi(pidStr); e2 == nil && oldPID > 0 && oldPID != os.Getpid() {
			if processAlive(oldPID) {
				_ = killProcess(oldPID)
				// 等端口/文件释放
				for i := 0; i < 20; i++ {
					if !processAlive(oldPID) {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
		_ = os.Remove(lockPath)
	}

	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return nil, fmt.Errorf("write lock: %w", err)
	}

	return func() {
		raw, e := os.ReadFile(lockPath)
		if e != nil {
			return
		}
		if strings.TrimSpace(string(raw)) == strconv.Itoa(os.Getpid()) {
			_ = os.Remove(lockPath)
		}
	}, nil
}
