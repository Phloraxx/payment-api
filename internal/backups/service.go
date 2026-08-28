package backups

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/alerts"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/archive"
	_ "modernc.org/sqlite"
)

type Service struct {
	App    core.App
	Config config.Config
	Alerts *alerts.Service
	Now    func() time.Time
}

type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

type Status struct {
	Enabled           bool      `json:"enabled"`
	Cron              string    `json:"cron,omitempty"`
	MaxKeep           int       `json:"maxKeep"`
	Offsite           bool      `json:"offsite"`
	BackupCount       int       `json:"backupCount"`
	Latest            *FileInfo `json:"latest,omitempty"`
	LatestVerified    bool      `json:"latestVerified"`
	VerificationError string    `json:"verificationError,omitempty"`
}

type RestoreDrillResult struct {
	BackupName       string   `json:"backupName"`
	DatabaseFiles    []string `json:"databaseFiles"`
	IntegrityChecked int      `json:"integrityChecked"`
}

func NewService(app core.App, cfg config.Config, alertService *alerts.Service) *Service {
	return &Service{App: app, Config: cfg, Alerts: alertService, Now: time.Now}
}

func (s *Service) Configure() error {
	settings := s.App.Settings()
	settings.Backups.Cron = s.Config.BackupCron
	settings.Backups.CronMaxKeep = s.Config.BackupMaxKeep
	settings.Backups.S3.Enabled = s.Config.BackupS3Enabled
	settings.Backups.S3.Bucket = s.Config.BackupS3Bucket
	settings.Backups.S3.Region = s.Config.BackupS3Region
	settings.Backups.S3.Endpoint = s.Config.BackupS3Endpoint
	settings.Backups.S3.AccessKey = s.Config.BackupS3AccessKey
	settings.Backups.S3.Secret = s.Config.BackupS3Secret
	settings.Backups.S3.ForcePathStyle = s.Config.BackupS3ForcePathStyle
	return s.App.Save(settings)
}

func (s *Service) RegisterHooks() {
	s.App.OnBackupCreate().BindFunc(func(event *core.BackupEvent) error {
		err := event.Next()
		if err != nil {
			if s.Alerts != nil {
				_, _, _ = s.Alerts.Open(alerts.Input{
					Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:create",
					Message: "PayGate backup creation or upload failed", Details: map[string]any{"name": event.Name, "error": err.Error()},
				})
			}
			return err
		}
		if s.Alerts != nil {
			_ = s.Alerts.Resolve("backup:create")
		}
		return nil
	})
}

func (s *Service) Create(ctx context.Context) (string, error) {
	name := "paygate_manual_" + s.now().Format("20060102_150405") + ".zip"
	if err := s.App.CreateBackup(ctx, name); err != nil {
		return "", err
	}
	return name, nil
}

func (s *Service) GetStatus(ctx context.Context, verify bool) (Status, error) {
	result := Status{
		Enabled: s.Config.BackupCron != "", Cron: s.Config.BackupCron,
		MaxKeep: s.Config.BackupMaxKeep, Offsite: s.Config.BackupS3Enabled,
	}
	files, err := s.list(ctx)
	if err != nil {
		return result, err
	}
	result.BackupCount = len(files)
	if len(files) == 0 {
		if verify {
			result.VerificationError = "no backups are available"
		}
		return result, nil
	}
	result.Latest = &files[0]
	if verify {
		if err := s.verify(ctx, files[0].Name); err != nil {
			result.VerificationError = err.Error()
			if s.Alerts != nil {
				_, _, _ = s.Alerts.Open(alerts.Input{
					Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:verify",
					Message: "Latest PayGate backup failed archive verification", Details: map[string]any{"name": files[0].Name, "error": err.Error()},
				})
			}
		} else {
			result.LatestVerified = true
			if s.Alerts != nil {
				_ = s.Alerts.Resolve("backup:verify")
			}
		}
	}
	return result, nil
}

func (s *Service) list(ctx context.Context) ([]FileInfo, error) {
	fsys, err := s.App.NewBackupsFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()
	fsys.SetContext(ctx)
	objects, err := fsys.List("")
	if err != nil {
		return nil, err
	}
	files := make([]FileInfo, 0, len(objects))
	for _, object := range objects {
		if object == nil || strings.HasSuffix(object.Key, "/") {
			continue
		}
		files = append(files, FileInfo{Name: object.Key, Size: object.Size, ModTime: object.ModTime.UTC()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ModTime.After(files[j].ModTime) })
	return files, nil
}

func (s *Service) verify(ctx context.Context, name string) error {
	path, cleanup, err := s.download(ctx, name)
	if err != nil {
		return err
	}
	defer cleanup()
	archive, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open backup archive: %w", err)
	}
	defer archive.Close()
	if len(archive.File) == 0 {
		return errors.New("backup archive is empty")
	}
	var hasDatabase bool
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(file.Name), ".db") {
			hasDatabase = true
		}
		entry, err := file.Open()
		if err != nil {
			return fmt.Errorf("open archive entry %s: %w", file.Name, err)
		}
		_, copyErr := io.Copy(io.Discard, entry)
		closeErr := entry.Close()
		if copyErr != nil {
			return fmt.Errorf("read archive entry %s: %w", file.Name, copyErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if !hasDatabase {
		return errors.New("backup archive contains no database file")
	}
	return nil
}

func (s *Service) RestoreDrill(ctx context.Context) (RestoreDrillResult, error) {
	files, err := s.list(ctx)
	if err != nil {
		return RestoreDrillResult{}, err
	}
	if len(files) == 0 {
		return RestoreDrillResult{}, errors.New("no backups are available")
	}
	path, cleanup, err := s.download(ctx, files[0].Name)
	if err != nil {
		return RestoreDrillResult{}, err
	}
	defer cleanup()
	destination, err := os.MkdirTemp("", "paygate-restore-drill-*")
	if err != nil {
		return RestoreDrillResult{}, err
	}
	defer os.RemoveAll(destination)
	if err := archive.Extract(path, destination); err != nil {
		return RestoreDrillResult{}, fmt.Errorf("extract backup: %w", err)
	}
	result := RestoreDrillResult{BackupName: files[0].Name}
	filesChecked, err := validateRestoredDatabases(ctx, destination)
	if err != nil {
		return RestoreDrillResult{}, err
	}
	result.DatabaseFiles = filesChecked
	result.IntegrityChecked = len(filesChecked)
	if result.IntegrityChecked == 0 {
		return RestoreDrillResult{}, errors.New("restored archive contains no SQLite database")
	}
	return result, nil
}

func validateRestoredDatabases(ctx context.Context, destination string) ([]string, error) {
	entries, err := os.ReadDir(destination)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, 2)
	for _, entry := range entries {
		// A PocketBase restore boots only the root database files. Nested .db files
		// are retained forensic/safety snapshots (for example quarantine/) and
		// must not make an otherwise restorable backup fail its drill.
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".db") {
			continue
		}
		path := filepath.Join(destination, entry.Name())
		database, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
		if err != nil {
			return nil, fmt.Errorf("open restored database %s: %w", entry.Name(), err)
		}
		var check string
		queryErr := database.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&check)
		closeErr := database.Close()
		if queryErr != nil {
			return nil, fmt.Errorf("integrity check %s: %w", entry.Name(), queryErr)
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if !strings.EqualFold(strings.TrimSpace(check), "ok") {
			return nil, fmt.Errorf("integrity check %s returned %q", entry.Name(), check)
		}
		files = append(files, entry.Name())
	}
	return files, nil
}

func (s *Service) download(ctx context.Context, name string) (string, func(), error) {
	fsys, err := s.App.NewBackupsFilesystem()
	if err != nil {
		return "", func() {}, err
	}
	fsys.SetContext(ctx)
	reader, err := fsys.GetReader(name)
	if err != nil {
		fsys.Close()
		return "", func() {}, err
	}
	temp, err := os.CreateTemp("", "paygate-backup-*.zip")
	if err != nil {
		reader.Close()
		fsys.Close()
		return "", func() {}, err
	}
	path := temp.Name()
	cleanup := func() {
		temp.Close()
		reader.Close()
		fsys.Close()
		os.Remove(path)
	}
	if _, err := io.Copy(temp, reader); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
