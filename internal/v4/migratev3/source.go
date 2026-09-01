package migratev3

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const maxSourceDataDBBytes int64 = 2 << 30

type sourceCopy struct {
	DB     *sql.DB
	Dir    string
	Path   string
	SHA256 string
}

func (s *sourceCopy) Close() {
	if s == nil {
		return
	}
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.Dir != "" {
		_ = os.RemoveAll(s.Dir)
	}
}
func openVerifiedSource(ctx context.Context, archive string) (*sourceCopy, error) {
	archive = strings.TrimSpace(archive)
	if archive == "" {
		return nil, errors.New("source backup ZIP is required")
	}
	abs, err := filepath.Abs(archive)
	if err != nil {
		return nil, fmt.Errorf("resolve source archive: %w", err)
	}
	digest, err := verifyArchiveSidecar(abs)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(abs)
	if err != nil {
		return nil, fmt.Errorf("open source backup ZIP: %w", err)
	}
	defer zr.Close()
	var dataFile *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case "data.db":
			if dataFile != nil {
				return nil, errors.New("source backup contains multiple data.db files")
			}
			dataFile = f
		case "data.db-wal":
			if f.UncompressedSize64 != 0 {
				return nil, errors.New("source backup contains a non-empty data.db-wal; use a checkpointed backup")
			}
		}
	}
	if dataFile == nil {
		return nil, errors.New("source backup does not contain data.db")
	}
	if dataFile.UncompressedSize64 == 0 || dataFile.UncompressedSize64 > uint64(maxSourceDataDBBytes) {
		return nil, errors.New("source data.db has an unsafe size")
	}
	dir, err := os.MkdirTemp("", "paygate-v3-migrate-*")
	if err != nil {
		return nil, fmt.Errorf("create source temp directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	outPath := filepath.Join(dir, "data.db")
	if err := extractDB(dataFile, outPath); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(outPath)+"?mode=ro&immutable=1")
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("open extracted source: %w", err)
	}
	s := &sourceCopy{DB: db, Dir: dir, Path: outPath, SHA256: digest}
	if err := verifySourceDB(ctx, db); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}
func verifyArchiveSidecar(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open source archive: %w", err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		_ = file.Close()
		return "", err
	}
	_ = file.Close()
	got := hex.EncodeToString(h.Sum(nil))
	sidecar := path + ".sha256"
	raw, err := os.ReadFile(sidecar)
	if err != nil {
		return "", fmt.Errorf("verified source requires %s: %w", sidecar, err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 || !strings.EqualFold(fields[0], got) {
		return "", errors.New("source backup SHA-256 sidecar does not match archive")
	}
	return got, nil
}

func extractDB(f *zip.File, destination string) error {
	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open data.db in backup: %w", err)
	}
	defer r.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create extracted data.db: %w", err)
	}
	written, copyErr := io.CopyN(out, r, maxSourceDataDBBytes+1)
	closeErr := out.Close()
	if copyErr != nil && !errors.Is(copyErr, io.EOF) {
		_ = os.Remove(destination)
		return copyErr
	}
	if written > maxSourceDataDBBytes {
		_ = os.Remove(destination)
		return errors.New("extracted data.db exceeds size limit")
	}
	if closeErr != nil {
		_ = os.Remove(destination)
		return closeErr
	}
	return nil
}
func verifySourceDB(ctx context.Context, db *sql.DB) error {
	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return fmt.Errorf("source integrity check: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(integrity), "ok") {
		return fmt.Errorf("source integrity check returned %q", integrity)
	}
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("source foreign key check: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("source database has foreign key violations")
	}
	for _, table := range []string{"payments", "relay_devices"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("source is missing required table %s", table)
		}
	}
	return nil
}
