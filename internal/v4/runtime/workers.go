package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (a *App) RunWorkers(ctx context.Context) {
	if a == nil {
		return
	}
	go a.Webhooks.Run(ctx)
	go a.expiryWorker(ctx)
	go a.backupWorker(ctx)
}

func (a *App) expiryWorker(ctx context.Context) {
	interval := a.Config.ExpiryInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			count, err := a.Payments.ExpireDue(ctx, 500)
			if err != nil {
				slog.Error("expire due payments", "error", err)
				continue
			}
			if count > 0 {
				a.Webhooks.Wake()
			}
		}
	}
}
func (a *App) BackupNow(ctx context.Context, at time.Time) (string, error) {
	if a == nil || a.DB == nil {
		return "", fmt.Errorf("PayGate storage is unavailable")
	}
	at = at.UTC()
	if err := os.MkdirAll(a.Config.BackupDir, 0o750); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}
	name := "paygate-" + at.Format("20060102T150405Z") + ".db"
	path := filepath.Join(a.Config.BackupDir, name)
	if err := a.DB.BackupTo(ctx, path); err != nil {
		return "", err
	}
	if err := pruneBackups(a.Config.BackupDir, a.Config.BackupRetention); err != nil {
		return path, err
	}
	return path, nil
}

func (a *App) backupWorker(ctx context.Context) {
	for {
		now := time.Now().UTC()
		next := nextDailyBackup(now, a.Config.BackupHourUTC)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			path, err := a.BackupNow(ctx, time.Now().UTC())
			if err != nil {
				slog.Error("PayGate backup failed", "error", err)
			} else {
				slog.Info("PayGate backup verified", "path", path)
			}
		}
	}
}
func nextDailyBackup(now time.Time, hourUTC int) time.Time {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func pruneBackups(dir string, keep int) error {
	if keep <= 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "paygate-") && strings.HasSuffix(name, ".db") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for len(names) > keep {
		path := filepath.Join(dir, names[0])
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove old backup %s: %w", names[0], err)
		}
		names = names[1:]
	}
	return nil
}
