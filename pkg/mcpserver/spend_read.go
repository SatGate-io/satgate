package mcpserver

import (
	"context"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// SpendUsage is a spending limit's counter for the current window, read
// without changing it. Units are 10^-8; Used and Remaining are the same
// numbers as plain decimals (the format of a refusal's "limit").
type SpendUsage struct {
	Used      string
	Remaining string
}

// SpendCounterKey is the store key of the counter a limit spends from for the
// window that contains now. It is the key the gate reserves under, so a read
// through it sees exactly what the gate counted.
func SpendCounterKey(tenant string, l toolcontrols.SpendLimit, now time.Time) string {
	return counterKey(tenant, counterName(l), windowIDOf(l, now))
}

// SpendTotalReader is the read half of toolcontrols.SpendStore.
type SpendTotalReader interface {
	Total(ctx context.Context, key string) (int64, error)
}

// ReadSpendUsage reads what a limit has used in the current window and what is
// left (max minus used, never negative).
func ReadSpendUsage(ctx context.Context, store SpendTotalReader, tenant string, l toolcontrols.SpendLimit, now time.Time) (SpendUsage, error) {
	total, err := store.Total(ctx, SpendCounterKey(tenant, l, now))
	if err != nil {
		return SpendUsage{}, err
	}
	return SpendUsage{Used: formatUnits(total), Remaining: formatUnits(leftUnits(l.MaxUnits(), total))}, nil
}
