package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// computeChecksum 计算文件内容的 SHA256
func computeChecksum(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func WatchConfig(ctx context.Context, configPath string, onChange func()) error {
	dir := filepath.Dir(configPath)
	fileName := filepath.Base(configPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// 初始 checksum（如果不存在，设为 nil）
	var lastChecksum []byte
	if _, err := os.Stat(configPath); err == nil {
		lastChecksum, _ = computeChecksum(configPath)
	}

	// 抖动定时器
	var debounceTimer *time.Timer
	const debounceDuration = 200 * time.Millisecond

	triggerCheck := func() {
		// 先确认文件存在再读
		if _, err := os.Stat(configPath); err != nil {
			// 文件暂时不存在（可能被替换），跳过，下一轮事件会再触发
			return
		}
		sum, err := computeChecksum(configPath)
		if err != nil {
			log.Printf("读取配置校验失败: %v", err)
			return
		}
		if lastChecksum == nil || !equal(sum, lastChecksum) {
			lastChecksum = sum
			// 真正变更，执行回调
			onChange()
		}
	}

	// 监听目录
	if err := watcher.Add(dir); err != nil {
		return err
	}

	log.Printf("开始监听配置文件: %s", configPath)

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher 事件通道关闭")
			}

			// 关心的文件相关事件（写、重命名、创建、删除、chmod 有时也表示替换）
			if filepath.Base(ev.Name) != fileName {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}

			// 防抖：收到事件后等待一小段时间再实际比对，合并频繁的连续事件
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, triggerCheck)

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher 错误通道关闭")
			}
			log.Printf("watcher 错误: %v", err)
		}
	}
}

// equal 比较两个 byte slice
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
