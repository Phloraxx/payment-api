package payments

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

var ErrPaymentCapacity = errors.New("payment capacity temporarily unavailable")

type RandomIndex func(max int) (int, error)

type Allocator struct {
	Random      RandomIndex
	SoftHorizon time.Duration
	Buckets     int
}

func NewAllocator() Allocator {
	return Allocator{
		Random:      cryptoRandomIndex,
		SoftHorizon: 4 * time.Hour,
		Buckets:     2,
	}
}

func (a Allocator) Select(ctx context.Context, tx *storage.ImmediateTx, profileID string, requestedAmountPaise int64, now time.Time) (int64, error) {
	if tx == nil {
		return 0, errors.New("sqlite connection is required")
	}
	if profileID == "" {
		return 0, errors.New("collection profile is required")
	}
	if requestedAmountPaise <= 0 || requestedAmountPaise%100 != 0 {
		return 0, errors.New("requested amount must be positive whole INR")
	}
	buckets := a.Buckets
	if buckets <= 0 {
		buckets = 2
	}
	randomIndex := a.Random
	if randomIndex == nil {
		randomIndex = cryptoRandomIndex
	}

	nowMS := now.UTC().UnixMilli()
	if _, err := tx.ExecContext(ctx, `UPDATE amount_reservations
        SET released_at = ?
        WHERE released_at IS NULL AND reserved_until <= ?`, nowMS, nowMS); err != nil {
		return 0, fmt.Errorf("release due amount reservations: %w", err)
	}

	cutoffMS := now.Add(-a.SoftHorizon).UTC().UnixMilli()
	for bucket := 0; bucket < buckets; bucket++ {
		start := requestedAmountPaise + int64(bucket*100) + 1
		end := requestedAmountPaise + int64(bucket*100) + 99
		candidates, err := loadBucketCandidates(ctx, tx, profileID, start, end, cutoffMS)
		if err != nil {
			return 0, err
		}
		if len(candidates) == 0 {
			continue
		}
		index, err := randomIndex(len(candidates))
		if err != nil {
			return 0, fmt.Errorf("choose payable amount: %w", err)
		}
		if index < 0 || index >= len(candidates) {
			return 0, fmt.Errorf("random index %d out of range for %d candidates", index, len(candidates))
		}
		return candidates[index], nil
	}
	return 0, ErrPaymentCapacity
}

func loadBucketCandidates(ctx context.Context, tx *storage.ImmediateTx, profileID string, start, end, softCutoffMS int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT payable_amount_paise,
       MAX(CASE WHEN released_at IS NULL THEN 1 ELSE 0 END) AS active,
       MAX(last_used_at) AS last_used_at
FROM amount_reservations
WHERE collection_profile_id = ?
  AND payable_amount_paise BETWEEN ? AND ?
GROUP BY payable_amount_paise`, profileID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query amount bucket: %w", err)
	}
	defer rows.Close()

	type use struct {
		active   bool
		lastUsed int64
	}
	used := make(map[int64]use, 99)
	for rows.Next() {
		var amount, lastUsed int64
		var active int
		if err := rows.Scan(&amount, &active, &lastUsed); err != nil {
			return nil, fmt.Errorf("scan amount bucket: %w", err)
		}
		used[amount] = use{active: active == 1, lastUsed: lastUsed}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate amount bucket: %w", err)
	}

	preferred := make([]int64, 0, 99)
	recent := make([]int64, 0, 99)
	for amount := start; amount <= end; amount++ {
		state, seen := used[amount]
		if seen && state.active {
			continue
		}
		if !seen || state.lastUsed < softCutoffMS {
			preferred = append(preferred, amount)
		} else {
			recent = append(recent, amount)
		}
	}
	if len(preferred) > 0 {
		return preferred, nil
	}
	return recent, nil
}

func cryptoRandomIndex(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("random range must be positive")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
