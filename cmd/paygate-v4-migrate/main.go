package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Phloraxx/payment-api/internal/v4/migratev3"
)

func main() {
	var opts migratev3.Options
	var reportPath string
	flag.StringVar(&opts.SourceZIP, "source-zip", "", "verified PocketBase backup ZIP (requires .sha256 sidecar)")
	flag.StringVar(&opts.Destination, "destination", "", "new v4 SQLite database path; must not exist")
	flag.StringVar(&opts.ActiveProfile, "active-profile", env("PAYGATE_V4_ACTIVE_PROFILE"), "kotak or paytm")
	flag.StringVar(&opts.KotakUPIID, "kotak-upi", env("PAYGATE_V4_KOTAK_UPI_ID"), "Kotak destination UPI ID")
	flag.StringVar(&opts.KotakPayee, "kotak-payee", env("PAYGATE_V4_KOTAK_PAYEE"), "Kotak payee label")
	flag.StringVar(&opts.PaytmUPIID, "paytm-upi", env("PAYGATE_V4_PAYTM_UPI_ID"), "Paytm destination UPI ID")
	flag.StringVar(&opts.PaytmPayee, "paytm-payee", env("PAYGATE_V4_PAYTM_PAYEE"), "Paytm payee label")
	flag.StringVar(&reportPath, "report", "", "optional JSON report path; must not exist")
	flag.Parse()

	report, err := migratev3.Run(context.Background(), opts)
	if err != nil {
		fatal(err)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(raw))
	if strings.TrimSpace(reportPath) != "" {
		if err := writeExclusive(reportPath, append(raw, '\n')); err != nil {
			fatal(err)
		}
	}
}
func env(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func writeExclusive(path string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "paygate-v4-migrate:", err)
	os.Exit(1)
}
