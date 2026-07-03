package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu     sync.Mutex
	file   *os.File
	writer io.Writer = os.Stdout
)

func Init(logPath string) error {
	mu.Lock()
	defer mu.Unlock()

	if logPath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	file = f
	writer = io.MultiWriter(os.Stdout, f)
	return nil
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		err := file.Close()
		file = nil
		writer = os.Stdout
		return err
	}
	return nil
}

func Printf(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	prefix := time.Now().Format("2006-01-02 15:04:05") + " "
	fmt.Fprintf(writer, prefix+format, args...)
}
