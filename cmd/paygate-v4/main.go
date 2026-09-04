package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	v4runtime "github.com/Phloraxx/payment-api/internal/v4/runtime"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		return healthcheck()
	}
	cfg, err := configFromEnv()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	app, err := v4runtime.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.Close()
	app.RunWorkers(ctx)

	server := &http.Server{
		Addr: app.Config.ListenAddr, Handler: app.Handler(),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("PayGate v4 listening on %s", app.Config.ListenAddr)
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown PayGate v4 HTTP server: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve PayGate v4: %w", err)
	}
}

func healthcheck() error {
	addr := strings.TrimSpace(os.Getenv("PAYGATE_V4_LISTEN_ADDR"))
	if addr == "" {
		addr = ":8091"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	} else if strings.HasPrefix(addr, "0.0.0.0:") {
		addr = "127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	} else if strings.HasPrefix(addr, "[::]:") {
		addr = "127.0.0.1:" + strings.TrimPrefix(addr, "[::]:")
	}
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Get("http://" + addr + "/health")
	if err != nil {
		return fmt.Errorf("PayGate v4 healthcheck: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("PayGate v4 healthcheck returned HTTP %d", response.StatusCode)
	}
	return nil
}

func configFromEnv() (v4runtime.Config, error) {
	hour := 3
	if value := strings.TrimSpace(os.Getenv("PAYGATE_V4_BACKUP_HOUR_UTC")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return v4runtime.Config{}, fmt.Errorf("invalid PAYGATE_V4_BACKUP_HOUR_UTC: %w", err)
		}
		hour = parsed
	}
	retention := 30
	if value := strings.TrimSpace(os.Getenv("PAYGATE_V4_BACKUP_RETENTION")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return v4runtime.Config{}, fmt.Errorf("invalid PAYGATE_V4_BACKUP_RETENTION: %w", err)
		}
		retention = parsed
	}
	origins := splitCSV(os.Getenv("PAYGATE_V4_ALLOWED_ORIGINS"))
	return v4runtime.Config{
		DataDir:                  os.Getenv("PAYGATE_V4_DATA_DIR"),
		ListenAddr:               os.Getenv("PAYGATE_V4_LISTEN_ADDR"),
		PublicURL:                os.Getenv("PAYGATE_V4_PUBLIC_URL"),
		AllowedOrigins:           origins,
		BootstrapAdminPassword:   os.Getenv("PAYGATE_V4_ADMIN_PASSWORD"),
		BootstrapMerchantAPIKey:  os.Getenv("PAYGATE_V4_MERCHANT_API_KEY"),
		BootstrapWebhookEndpoint: os.Getenv("PAYGATE_V4_WEBHOOK_ENDPOINT"),
		BootstrapWebhookSecret:   os.Getenv("PAYGATE_V4_WEBHOOK_SECRET"),
		BackupDir:                os.Getenv("PAYGATE_V4_BACKUP_DIR"),
		BackupHourUTC:            hour,
		BackupRetention:          retention,
	}, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}
