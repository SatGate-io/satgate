package argrules

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules/internal/releasedv3"
)

func TestShareSellValuesLikeBuy(t *testing.T) {
	for _, side := range []string{"sell", "SELL", "Sell"} {
		for _, typ := range []string{"limit", "stop_limit"} {
			for _, hours := range []string{"-", `"regular_hours"`, `"extended_hours"`} {
				for _, size := range []struct {
					qty, price string
					units      int64
				}{
					{`"3"`, `"10"`, 3000000000},
					{`"0.5"`, `"199.99"`, 9999500000},
					{`2`, `49.5`, 9900000000},
					{`"1.00000001"`, `"1.00000001"`, 100000003},
				} {
					t.Run(fmt.Sprintf("%s/%s/%s/%s", side, typ, hours, size.qty), func(t *testing.T) {
						over := []string{"type", `"` + typ + `"`, "market_hours", hours, "quantity", size.qty, "limit_price", size.price}
						buy := mustUnits(t, "place_equity_order", dayBuy(t, over...))
						sell := mustUnits(t, "place_equity_order", dayBuy(t, append(over, "side", `"`+side+`"`)...))
						if buy != size.units || sell != buy {
							t.Fatalf("buy=%d sell=%d want=%d", buy, sell, size.units)
						}
					})
				}
			}
		}
	}
}

func TestShareSellRefusalsLikeBuy(t *testing.T) {
	for _, tc := range []struct {
		name string
		over []string
		why  OrderCostWhy
	}{
		{"market", []string{"type", `"market"`}, WhyNotALimitBuy},
		{"stop", []string{"type", `"stop"`}, WhyNotALimitBuy},
		{"stop_market", []string{"type", `"stop_market"`}, WhyNotALimitBuy},
		{"gtc", []string{"time_in_force", `"gtc"`}, WhyNotADayOrder},
		{"no_tif", []string{"time_in_force", "-"}, WhyNoTimeInForce},
		{"overnight", []string{"market_hours", `"all_day_hours"`}, WhyOvernightSession},
		{"no_price", []string{"limit_price", "-"}, WhyNoLimitPrice},
		{"both_sizes", []string{"dollar_amount", `"5"`}, WhyBothSizes},
		{"neither_size", []string{"quantity", "-"}, WhyNoSize},
		{"case_twin", []string{"Quantity", `"2"`}, WhyUnreadable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, side := range []string{`"buy"`, `"sell"`} {
				c, why := OrderMaxCost("place_equity_order", dayBuy(t, append(tc.over, "side", side)...))
				if c.Units != 0 || why != tc.why {
					t.Fatalf("%s: %v %q want %q", side, c, why, tc.why)
				}
			}
		})
	}
	for _, field := range []string{"quantity", "limit_price"} {
		for _, tc := range []struct {
			raw string
			why OrderCostWhy
		}{
			{`0`, WhyNotAboveZero}, {`"-1"`, WhyNotAboveZero},
			{`"1e3"`, WhyBadNumber}, {`null`, WhyBadNumber}, {`true`, WhyBadNumber},
			{`"10000000000"`, WhyCostTooLarge},
		} {
			t.Run(field+"/"+tc.raw, func(t *testing.T) {
				for _, side := range []string{`"buy"`, `"sell"`} {
					c, why := OrderMaxCost("place_equity_order", dayBuy(t, "side", side, field, tc.raw))
					if c.Units != 0 || why != tc.why {
						t.Fatalf("%s: %v %q want %q", side, c, why, tc.why)
					}
				}
			})
		}
	}
	for _, side := range []string{"-", `null`, `1`, `"short"`, `"sell_short"`, `"buy_to_cover"`, `"ſell"`} {
		t.Run("side/"+side, func(t *testing.T) {
			if c, why := OrderMaxCost("place_equity_order", dayBuy(t, "side", side)); c.Units != 0 || why != WhyNotABuy {
				t.Fatalf("%v %q", c, why)
			}
		})
	}
}

// The frozen pre-event version's order-cost implementation is byte-identical
// to main's (apart from its package/header). Keep all non-share-equity results.
func TestShareSellDollarsAndCryptoUnchanged(t *testing.T) {
	for _, tool := range []string{"place_equity_order", "place_crypto_order"} {
		for _, side := range []string{`"buy"`, `"sell"`} {
			for _, raw := range []string{`"20"`, `2.5`, `"0.000000011"`, `0`, `-1`, `null`, `"junk"`, `"10000000000"`} {
				for _, extra := range [][]string{nil, {"limit_price", `"5"`}, {"stop_price", `null`}, {"quantity", `"1"`}, {"Dollar_Amount", `"2"`}} {
					a := args(t, append([]string{"side", side, "dollar_amount", raw}, extra...)...)
					c, why := OrderMaxCost(tool, a)
					old, oldWhy := releasedv3.OrderMaxCost(tool, a)
					if c.Units != old.Units || string(why) != string(oldWhy) {
						t.Fatalf("dollar result changed: %v %q vs %v %q", c, why, old, oldWhy)
					}
				}
			}
			if tool == "place_crypto_order" {
				for _, typ := range []string{`"market"`, `"limit"`, `"stop_limit"`} {
					for _, tif := range []string{"-", `"gfd"`, `"gtc"`} {
						a := dayBuy(t, "side", side, "type", typ, "time_in_force", tif)
						c, why := OrderMaxCost(tool, a)
						old, oldWhy := releasedv3.OrderMaxCost(tool, a)
						if c.Units != old.Units || string(why) != string(oldWhy) {
							t.Fatalf("crypto result changed: %v %q vs %v %q", c, why, old, oldWhy)
						}
					}
				}
			}
		}
	}
}

func TestShareSellApprovedWords(t *testing.T) {
	for _, tc := range []struct{ got, want string }{
		{OrderCostCountedAs, "dollar_amount, or quantity times limit_price for a day limit order of shares"},
		{string(WhyNotALimitBuy), "an order by quantity is only counted for a limit or stop-limit order; market and stop orders by quantity cannot be counted"},
		{string(WhyNoLimitPrice), "a limit order by quantity must give limit_price"},
		{string(WhyNotABuy), "an order by quantity is only counted when side is buy or sell"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q want %q", tc.got, tc.want)
		}
	}
	// Also exercise the exact tools/call reader with a sell, not just maps.
	p := json.RawMessage(`{"name":"place_equity_order","arguments":{"side":"sell","type":"limit","time_in_force":"gfd","quantity":"2","limit_price":"10"}}`)
	if c, why := OrderMaxCostOfCall(p, "place_equity_order", "place_equity_order"); c.Units != 2000000000 || !why.Valued() {
		t.Fatalf("%v %q", c, why)
	}
}
