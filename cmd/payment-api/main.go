package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Phloraxx/payment-api/internal/alerts"
	"github.com/Phloraxx/payment-api/internal/api"
	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/backups"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/razorpaytest"
	"github.com/Phloraxx/payment-api/internal/reconciliation"
	"github.com/Phloraxx/payment-api/internal/refunds"
	"github.com/Phloraxx/payment-api/internal/retention"
	"github.com/Phloraxx/payment-api/internal/reviews"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/Phloraxx/payment-api/internal/webhooks"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:  filepath.Clean(cfg.DataDir),
		HideStartBanner: false,
	})
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{Automigrate: false})

	zeroLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}).With().Timestamp().Logger()
	gmessagesLogger := gmessages.ProductionLogger(zeroLogger)
	stdLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	webhookService := webhooks.NewService(app, cfg)
	webhookService.Logger = stdLogger
	paymentService := payments.NewService(app, cfg, webhookService)
	auditService := audit.NewService(app)
	alertService := alerts.NewService(app)
	alertService.ConfigureWebhook(cfg.OperatorAlertWebhookURL, cfg.OperatorAlertWebhookSecret)
	reviewService := reviews.NewService(app, paymentService, auditService)
	smsService := sms.NewService(app, paymentService)
	smsService.Reviews = reviewService
	reconciliationService := reconciliation.NewService(app, reviewService, alertService, auditService)
	statementLocation, err := time.LoadLocation(cfg.StatementTimezone)
	if err != nil {
		log.Fatal(err)
	}
	reconciliationService.StatementLocation = statementLocation
	refundService := refunds.NewService(app, auditService, webhookService)
	var razorpayTestService *razorpaytest.Service
	if cfg.RazorpayTestEnabled {
		razorpayClient := razorpaytest.NewClient(cfg.RazorpayTestKeyID, cfg.RazorpayTestKeySecret)
		razorpayTestService = razorpaytest.NewService(app, razorpayClient, cfg.RazorpayTestKeyID, cfg.RazorpayTestKeySecret, cfg.RazorpayTestWebhookSecret, cfg.RazorpayTestDisplayName)
	}
	retentionService := retention.NewService(app, cfg)
	backupService := backups.NewService(app, cfg, alertService)
	backupService.RegisterHooks()
	gmessagesManager := gmessages.NewManager(cfg, gmessagesLogger, func(input sms.Input) error {
		_, err := smsService.Ingest(input)
		return err
	})
	apiService := api.New(cfg, paymentService, smsService, gmessagesManager)
	apiService.Reviews = reviewService
	apiService.Reconciliation = reconciliationService
	apiService.Alerts = alertService
	apiService.Refunds = refundService
	apiService.Backups = backupService
	apiService.RazorpayTest = razorpayTestService
	apiService.Register(app)
	registerPairCommand(app, cfg, gmessagesLogger)
	registerHealthcheckCommand(app)
	registerBackupCommands(app, backupService)

	rootCtx, rootCancel := context.WithCancel(context.Background())
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		rateLimits := &e.App.Settings().RateLimits
		rateLimits.Enabled = cfg.RateLimitsEnabled
		rateLimits.Rules = mergeManagedRateLimitRules(rateLimits.Rules)
		if err := cfg.ValidateServe(); err != nil {
			return err
		}
		if err := backupService.Configure(); err != nil {
			return fmt.Errorf("configure backups: %w", err)
		}
		if cfg.LegacySMSWebhookEnabled {
			stdLogger.Warn("legacy /api/webhook compatibility route is enabled; rotate the old relay to SMS_WEBHOOK_SECRET and disable it")
		}
		startBackgroundRunners(rootCtx, webhookService, alertService)
		gmessagesManager.Start(rootCtx)
		return e.Next()
	})
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		rootCancel()
		gmessagesManager.Stop()
		return e.Next()
	})

	app.Cron().MustAdd("paygate-expire-payments", "* * * * *", func() {
		count, err := paymentService.ExpireDue()
		if err != nil {
			stdLogger.Error("payment expiry job failed", "error", err)
			return
		}
		if count > 0 {
			stdLogger.Info("expired due payments", "count", count)
		}
	})
	app.Cron().MustAdd("paygate-webhook-retries", "* * * * *", func() {
		webhookService.Wake()
	})
	app.Cron().MustAdd("paygate-operator-alert-deliveries", "* * * * *", func() { alertService.Wake() })
	app.Cron().MustAdd("paygate-operational-alerts", "* * * * *", func() {
		if err := alertService.CheckConnector(gmessagesManager.Status()); err != nil {
			stdLogger.Error("connector alert check failed", "error", err)
		}
		capacity, err := paymentService.Capacity()
		if err != nil {
			stdLogger.Error("capacity alert calculation failed", "error", err)
		} else if err := alertService.CheckCapacity(capacity); err != nil {
			stdLogger.Error("capacity alert check failed", "error", err)
		}
		if err := alertService.CheckWebhookExhaustion(); err != nil {
			stdLogger.Error("webhook exhaustion alert check failed", "error", err)
		}
	})
	app.Cron().MustAdd("paygate-retention", "17 2 * * *", func() {
		result, err := retentionService.Run()
		if err != nil {
			stdLogger.Error("retention job failed", "error", err)
			return
		}
		if result.SMSEventsRedacted+result.ReconciliationEntriesRedacted+result.AuditEventsDeleted > 0 {
			stdLogger.Info("retention job completed", "smsRedacted", result.SMSEventsRedacted, "reconciliationRedacted", result.ReconciliationEntriesRedacted, "auditDeleted", result.AuditEventsDeleted)
		}
	})
	app.Cron().MustAdd("paygate-backup-verify", "23 4 * * *", func() {
		if cfg.BackupCron == "" {
			return
		}
		status, err := backupService.GetStatus(context.Background(), true)
		if err != nil {
			stdLogger.Error("backup verification failed", "error", err)
			return
		}
		if !status.LatestVerified {
			stdLogger.Error("latest backup is not verified", "error", status.VerificationError)
		}
	})
	app.Cron().MustAdd("paygate-restore-drill", "41 4 1 * *", func() {
		if cfg.BackupCron == "" {
			return
		}
		result, err := backupService.RestoreDrill(context.Background())
		if err != nil {
			stdLogger.Error("monthly backup restore drill failed", "error", err)
			_, _, _ = alertService.Open(alerts.Input{Kind: "backup_failed", Severity: "critical", DedupeKey: "backup:restore-drill", Message: "Monthly backup restore drill failed", Details: map[string]any{"error": err.Error()}})
			return
		}
		_ = alertService.Resolve("backup:restore-drill")
		stdLogger.Info("monthly backup restore drill passed", "backup", result.BackupName, "databases", result.IntegrityChecked)
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func registerPairCommand(app *pocketbase.PocketBase, cfg config.Config, logger zerolog.Logger) {
	var qrPNG string
	cmd := &cobra.Command{
		Use:   "gmessages-pair",
		Short: "Pair the optional Google Messages connector",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return gmessages.PairConsole(ctx, cfg.GMessagesSessionPath, qrPNG, logger)
		},
	}
	cmd.Flags().StringVar(&qrPNG, "qr-png", "", "also write/refresh the pairing QR as a PNG file")
	app.RootCmd.AddCommand(cmd)
}

func registerHealthcheckCommand(app *pocketbase.PocketBase) {
	var endpoint string
	cmd := &cobra.Command{
		Use:    "healthcheck",
		Short:  "Check a running PayGate/PocketBase HTTP server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := &http.Client{Timeout: 3 * time.Second}
			response, err := client.Get(endpoint)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				return fmt.Errorf("health endpoint returned HTTP %d", response.StatusCode)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&endpoint, "url", "http://127.0.0.1:3000/api/health", "health endpoint URL")
	app.RootCmd.AddCommand(cmd)
}

type backgroundRunner interface {
	Run(context.Context)
}

func startBackgroundRunners(ctx context.Context, runners ...backgroundRunner) {
	for _, runner := range runners {
		if runner != nil {
			go runner.Run(ctx)
		}
	}
}

func mergeManagedRateLimitRules(existing []core.RateLimitRule) []core.RateLimitRule {
	managed := []core.RateLimitRule{
		{Label: "POST /api/events/sms", MaxRequests: 60, Duration: 60},
		{Label: "POST /api/webhook", MaxRequests: 30, Duration: 60},
		{Label: "POST /api/payments", MaxRequests: 120, Duration: 60},
		{Label: "POST /api/razorpay/test/orders", MaxRequests: 30, Duration: 60},
		{Label: "POST /api/razorpay/test/webhook", MaxRequests: 120, Duration: 60},
	}
	labels := make(map[string]struct{}, len(managed))
	for _, rule := range managed {
		labels[rule.Label] = struct{}{}
	}
	result := append([]core.RateLimitRule(nil), managed...)
	for _, rule := range existing {
		if _, controlled := labels[rule.Label]; controlled {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func registerBackupCommands(app *pocketbase.PocketBase, service *backups.Service) {
	createCmd := &cobra.Command{
		Use:   "backup-create",
		Short: "Create a PayGate/PocketBase backup now",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := service.Create(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), name)
			return nil
		},
	}
	verifyCmd := &cobra.Command{
		Use:   "backup-verify",
		Short: "Download and verify the latest PayGate backup archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := service.GetStatus(cmd.Context(), true)
			if err != nil {
				return err
			}
			if !status.LatestVerified {
				return fmt.Errorf("backup verification failed: %s", status.VerificationError)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "verified %s (%d bytes)\n", status.Latest.Name, status.Latest.Size)
			return nil
		},
	}
	restoreDrillCmd := &cobra.Command{
		Use:   "backup-restore-drill",
		Short: "Restore the latest backup into a temporary directory and run database integrity checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.RestoreDrill(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "restore drill passed for %s (%d database files)\n", result.BackupName, result.IntegrityChecked)
			return nil
		},
	}
	app.RootCmd.AddCommand(createCmd, verifyCmd, restoreDrillCmd)
}
