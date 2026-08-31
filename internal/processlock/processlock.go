package processlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const filename = ".paygate-process.lock"

type Lock struct {
	file *os.File
}

func Acquire(dataDir string) (*Lock, error) {
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	path := filepath.Join(dataDir, filename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open process lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, fmt.Errorf("PayGate data directory %q is already owned by another process", dataDir)
		}
		return nil, fmt.Errorf("acquire process lock: %w", err)
	}
	return &Lock{file: file}, nil
}

func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	if err != nil {
		return fmt.Errorf("release process lock: %w", err)
	}
	return closeErr
}
