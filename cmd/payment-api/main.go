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

	"github.com/Phloraxx/payment-api/internal/api"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/payments"
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
	smsService := sms.NewService(app, paymentService)
	gmessagesManager := gmessages.NewManager(cfg, gmessagesLogger, func(input sms.Input) error {
		_, err := smsService.Ingest(input)
		return err
	})
	api.New(cfg, paymentService, smsService, gmessagesManager).Register(app)
	registerPairCommand(app, cfg, gmessagesLogger)
	registerHealthcheckCommand(app)

	rootCtx, rootCancel := context.WithCancel(context.Background())
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		rateLimits := &e.App.Settings().RateLimits
		rateLimits.Enabled = cfg.RateLimitsEnabled
		rateLimits.Rules = append([]core.RateLimitRule{
			{Label: "POST /api/events/sms", MaxRequests: 60, Duration: 60},
			{Label: "POST /api/webhook", MaxRequests: 30, Duration: 60},
			{Label: "POST /api/payments", MaxRequests: 120, Duration: 60},
		}, rateLimits.Rules...)
		if err := cfg.ValidateServe(); err != nil {
			return err
		}
		if cfg.LegacySMSWebhookEnabled {
			stdLogger.Warn("legacy /api/webhook compatibility route is enabled; rotate the old relay to SMS_WEBHOOK_SECRET and disable it")
		}
		go webhookService.Run(rootCtx)
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
