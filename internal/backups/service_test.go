package backups

import (
	"context"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func TestConfigureCreateAndVerifyLocalBackup(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	cfg := config.Config{BackupCron: "0 3 * * *", BackupMaxKeep: 7}
	service := NewService(app, cfg, nil)
	service.Now = func() time.Time { return time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC) }
	if err := service.Configure(); err != nil {
		t.Fatal(err)
	}
	if app.Settings().Backups.Cron != cfg.BackupCron || app.Settings().Backups.CronMaxKeep != 7 {
		t.Fatalf("backup settings=%+v", app.Settings().Backups)
	}
	name, err := service.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if name != "paygate_manual_20260801_080000.zip" {
		t.Fatalf("name=%s", name)
	}
	status, err := service.GetStatus(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if status.BackupCount != 1 || status.Latest == nil || !status.LatestVerified || status.VerificationError != "" {
		t.Fatalf("status=%+v", status)
	}
	drill, err := service.RestoreDrill(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if drill.BackupName != name || drill.IntegrityChecked == 0 || len(drill.DatabaseFiles) == 0 {
		t.Fatalf("drill=%+v", drill)
	}
}
