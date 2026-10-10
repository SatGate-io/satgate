package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// DenialDuplicateRequest is the code for a call that repeats a request id the
// gate already admitted, with the same params, in the same window. Forwarding
// it again would send a second order that the limit counts once.
const DenialDuplicateRequest = "DUPLICATE_REQUEST"

// counterGrace keeps a counter a little past the end of its window, so a
// settle that runs just after the boundary still finds it.
const counterGrace = time.Hour

// SpendGate is the CallGate that enforces a token's spending limits
// (toolcontrols.SpendLimit) against a durable SpendStore.
//
// For each limit that covers the call it reads the limit's field as an exact
// decimal and reserves it in the store before the call is forwarded. If the
// total would pass the owner's maximum, the call is refused. When the call
// ends, the reservation is kept if the upstream accepted the call and given
// back if it did not (a transport failure, a JSON-RPC error, an isError
// result), so only orders that went through count.
type SpendGate struct {
	store toolcontrols.SpendStore

	// CallScope, when set, narrows what "the same call" means (for example
	// to one client session). Two calls with the same request id and params
	// but a different scope are different calls.
	CallScope func(ctx context.Context) string

	// now is the clock; nil means time.Now. Tests set it.
	now func() time.Time
}

// NewSpendGate returns a gate over store. A nil store refuses every covered
// call (fail closed).
func NewSpendGate(store toolcontrols.SpendStore) *SpendGate { return &SpendGate{store: store} }

// held is one reservation the gate took for the call.
type held struct {
	res   toolcontrols.Reservation
	limit toolcontrols.SpendLimit
}

// Admit implements CallGate.
func (g *SpendGate) Admit(ctx context.Context, call GateCall) (*GateAdmission, error) {
	limits, err := call.Token.SpendLimitsFor(call.Tool)
	if err != nil {
		return nil, fmt.Errorf("token controls: %w", err)
	}
	if len(limits) == 0 {
		return nil, nil
	}
	if g == nil || g.store == nil {
		return nil, fmt.Errorf("no spend limit store is configured")
	}
	now := call.Now
	if g.now != nil {
		now = g.now()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	callID := g.callID(ctx, call)
	var taken []held
	giveBack := func() {
		for i := len(taken) - 1; i >= 0; i-- {
			g.release(ctx, taken[i].res)
		}
	}
	var first held
	var firstTotal int64

	for _, l := range limits {
		deny := func(msg string) *GateAdmission {
			giveBack()
			return &GateAdmission{Refusal: spendRefusal(call.Tool, l, msg)}
		}
		if l.ID == "" {
			// A limit with no identity has no counter. Never skip it.
			giveBack()
			return nil, fmt.Errorf("a spending limit has no counter id")
		}
		amount, why := argrules.ReadNumericField(call.Params, call.Tool, l.Field)
		if why != argrules.NumericFieldOK {
			return deny(fmt.Sprintf("%s: this token has a limit of %s on %s, and %s, so the limit cannot be checked",
				call.Tool, limitPhrase(l), l.Field, fieldReason(why))), nil
		}
		units, ok := amountUnits(amount)
		if !ok {
			return deny(fmt.Sprintf("%s: this token's limit is %s on %s; this order would go over it",
				call.Tool, limitPhrase(l), l.Field)), nil
		}
		windowID, _, end := l.WindowBounds(now)
		res := toolcontrols.Reservation{
			Key:       counterKey(call.Token.TenantID, l.ID, windowID),
			CallID:    callID,
			Units:     units,
			Max:       l.MaxUnits(),
			ExpiresAt: end.Add(counterGrace),
		}
		out, err := g.store.Reserve(ctx, res)
		if err != nil {
			giveBack()
			return nil, fmt.Errorf("spend limit store: %w", err)
		}
		switch out.Status {
		case toolcontrols.OverLimit:
			return deny(fmt.Sprintf("%s: this token's limit is %s on %s; this order would go over it",
				call.Tool, limitPhrase(l), l.Field)), nil
		case toolcontrols.Duplicate:
			// The earlier call with this id holds a reservation. Do not
			// release it from here, and do not forward a second copy.
			giveBack()
			return &GateAdmission{Refusal: &GateRefusal{
				Code:    DenialDuplicateRequest,
				Message: fmt.Sprintf("%s: this request id was already used for the same call; send it with a new id to try again", call.Tool),
			}}, nil
		}
		h := held{res: res, limit: l}
		if len(taken) == 0 {
			first, firstTotal = h, out.Total
		}
		taken = append(taken, h)
	}

	detail := GateReceiptDetail{
		ArgumentField: first.limit.Field,
		Window:        string(first.limit.Window),
		Limit:         first.limit.Max,
		WindowTotal:   formatUnits(firstTotal),
	}
	return &GateAdmission{
		Detail: detail,
		Settle: func(ctx context.Context, o GateOutcome) {
			if o.Succeeded {
				return // the amount stays counted
			}
			giveBack()
		},
	}, nil
}

func (g *SpendGate) callID(ctx context.Context, call GateCall) string {
	if call.RequestID == "" {
		// A request with no id cannot be recognised as a retry: each is its
		// own call.
		var n [16]byte
		if _, err := rand.Read(n[:]); err == nil {
			return hex.EncodeToString(n[:])
		}
	}
	h := sha256.New()
	h.Write([]byte(call.RequestID))
	h.Write([]byte{0})
	sum := sha256.Sum256(call.Params)
	h.Write(sum[:])
	if g.CallScope != nil {
		h.Write([]byte{0})
		h.Write([]byte(g.CallScope(ctx)))
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// release gives a reservation back. It runs on a context that survives the
// caller going away, because the call has ended either way.
func (g *SpendGate) release(ctx context.Context, r toolcontrols.Reservation) {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := g.store.Release(rctx, r); err != nil {
		// The counter stays higher than the real spend until the window ends:
		// the safe direction. Say so loudly.
		log.Error().Err(err).Str("counter", r.Key).Msg("could not release a spend reservation; the window total stays counted")
	}
}

func spendRefusal(tool string, l toolcontrols.SpendLimit, message string) *GateRefusal {
	return &GateRefusal{
		Code:    toolcontrols.DenialCodeSpendLimit,
		Message: message,
		Data: map[string]interface{}{
			"field":  l.Field,
			"window": string(l.Window),
			"limit":  l.Max,
		},
		Detail: GateReceiptDetail{
			DenialCode:    toolcontrols.DenialCodeSpendLimit,
			ArgumentField: l.Field,
			Window:        string(l.Window),
			Limit:         l.Max,
		},
	}
}

func limitPhrase(l toolcontrols.SpendLimit) string {
	return fmt.Sprintf("%s %s", toolcontrols.FormatAmount(l.Field, l.Max), l.Window.Phrase())
}

func fieldReason(why string) string {
	switch why {
	case argrules.NumericFieldMissing:
		return "the call has no value for it"
	case argrules.NumericFieldNotNumber:
		return "the value is not a plain number"
	}
	return "the call could not be read"
}

func counterKey(tenant, limitID, windowID string) string {
	return "satgate:spendlimit:" + tenant + ":" + limitID + ":" + windowID
}

// amountUnits is the amount in 10^-8 units, rounded up so it is never
// undercounted. A negative amount, or one too large to count, is not accepted.
func amountUnits(d *argrules.Decimal) (int64, bool) {
	r := d.Rat()
	if r.Sign() < 0 {
		return 0, false
	}
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt64(100000000))
	q, rem := new(big.Int).QuoRem(scaled.Num(), scaled.Denom(), new(big.Int))
	if rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, false
	}
	return q.Int64(), true
}

// formatUnits writes 10^-8 units as a plain decimal with no trailing zeros.
func formatUnits(u int64) string {
	whole, frac := u/100000000, u%100000000
	if frac == 0 {
		return fmt.Sprintf("%d", whole)
	}
	return fmt.Sprintf("%d.%s", whole, strings.TrimRight(fmt.Sprintf("%08d", frac), "0"))
}
