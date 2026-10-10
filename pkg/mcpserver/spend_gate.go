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
// total would pass the owner's maximum, the call is refused.
//
// When the call ends, the reservation is given back only when SatGate knows
// the call did not run upstream:
//
//   - it was never sent (a dial or connect failure before any byte of the
//     request was written, a request that could not be built, a refusal by
//     a later gate, a session closed before dispatch), or
//   - the upstream answered with a JSON-RPC protocol error that by the
//     JSON-RPC 2.0 specification means the request was not processed:
//     -32700 (parse error), -32600 (invalid request), -32601 (method not
//     found), -32602 (invalid params).
//
// It is kept for everything else: the amount stays counted when the upstream
// accepted the call, and also when the call was sent and SatGate cannot tell
// whether it ran: any other JSON-RPC error (-32603 Internal error,
// -32000..-32099, any other code), a result with isError=true (a tool can
// fail after a partial or full execution), a timeout after the send, a
// connection reset or closed after the send, an unreadable or partial reply,
// a context ended after dispatch, an HTTP error status with no JSON-RPC body,
// a redirect of the call. An order that may have been placed must stay
// counted, or the agent could spend past the owner's maximum. Those are
// logged as "outcome unknown; reservation kept".
//
// Known effect, accepted: an order the broker itself rejects (insufficient
// funds, market closed) still counts toward the day's total.
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
	total int64 // the counter after this reservation
}

// Admit implements CallGate.
func (g *SpendGate) Admit(ctx context.Context, call GateCall) (*GateAdmission, error) {
	all, err := call.Token.AllSpendLimits()
	if err != nil {
		return nil, fmt.Errorf("token controls: %w", err)
	}
	if len(all) == 0 {
		return nil, nil
	}
	// The token carries a limit, so the name this gate matches must be the
	// name the upstream reads. A name outside the plain set (spaces, Unicode
	// look-alikes, control characters) or a call whose top level reads two
	// ways (a repeated key, "Name" next to "name") could be one tool to the
	// upstream and another here, and it is refused for every tool on the
	// token, not only the covered ones. A case-variant of a plain name is
	// handled by SpendLimit.Covers, which ignores case.
	if !argrules.ValidToolName(call.Tool) || argrules.CheckCallShape(call.Params, call.Tool) != argrules.NumericFieldOK {
		return &GateAdmission{Refusal: spendRefusal(call.Tool, all[0],
			"this token has a spending limit, and this call's tool name or layout cannot be matched exactly to it, so the call was not sent")}, nil
	}
	var limits []toolcontrols.SpendLimit
	for _, l := range all {
		if l.Covers(call.Tool) {
			limits = append(limits, l)
		}
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

	// A limit that names a shared budget counts only when a budget_bind seal
	// covers the caveat that carries it. Anyone holding a token can append
	// caveats offline, so an unsealed reference is refused before any counter
	// is read or changed: it can neither spend from a budget nor read one.
	for _, l := range limits {
		if l.Budget != "" && (!l.Sealed || call.Token == nil || call.Token.TenantID == "") {
			return &GateAdmission{Refusal: spendRefusalFor(l, "",
				fmt.Sprintf("%s: this token refers to a shared budget that could not be confirmed, so the call was not sent", call.Tool), nil)}, nil
		}
	}

	callID := g.callID(ctx, call)
	var taken []held
	giveBack := func() {
		for i := len(taken) - 1; i >= 0; i-- {
			g.release(ctx, taken[i].res)
		}
	}
	var symbol string

	for _, l := range limits {
		sym := ""
		deny := func(msg string, remaining *int64) *GateAdmission {
			giveBack()
			return &GateAdmission{Refusal: spendRefusalFor(l, sym, msg, remaining)}
		}
		if l.ID == "" {
			// A limit with no identity has no counter. Never skip it.
			giveBack()
			return nil, fmt.Errorf("a spending limit has no counter id")
		}
		if len(l.Symbols) > 0 {
			raw, why := argrules.ReadStringField(call.Params, call.Tool, toolcontrols.SymbolField)
			verdict, canon := l.CheckSymbol(call.Tool, raw, why == argrules.NumericFieldOK)
			switch verdict {
			case toolcontrols.SymbolOther:
				continue // another symbol: this limit does not touch the call
			case toolcontrols.SymbolUnreadable:
				return deny(fmt.Sprintf("%s: this token caps spending by symbol, and the call has no %s the cap can read, so the limit cannot be checked; send the symbol as plain text such as BTC or BTC-USD",
					call.Tool, toolcontrols.SymbolField), nil), nil
			case toolcontrols.SymbolAmbiguous:
				// canon is the stem from the cap's own list, not the agent's text.
				return deny(fmt.Sprintf("%s: this token caps %s, and a symbol ending in USD with no hyphen could mean that coin or another; send the symbol as %q",
					call.Tool, canon, canon), nil), nil
			}
			sym = canon
		}
		key := counterKey(call.Token.TenantID, counterName(l), windowIDOf(l, now))
		amount, why := argrules.ReadNumericField(call.Params, call.Tool, l.Field)
		if why != argrules.NumericFieldOK {
			return deny(fmt.Sprintf("%s: this token has a limit of %s on %s, and %s, so the limit cannot be checked%s",
				call.Tool, limitPhrase(l), l.Field, fieldReason(why), sendFieldHint(l, why)), g.remainingAt(ctx, l, key)), nil
		}
		units, ok := amountUnits(amount)
		if !ok {
			rem := g.remainingAt(ctx, l, key)
			return deny(overMessage(call.Tool, l, sym, rem), rem), nil
		}
		_, _, end := l.WindowBounds(now)
		res := toolcontrols.Reservation{
			Key:       key,
			CallID:    callID,
			Units:     units,
			Max:       l.MaxUnits(),
			ExpiresAt: end.Add(counterGrace),
		}
		// Two limits of one call can name one counter (the same budget twice).
		// It is counted once; limits that disagree on the maximum are refused.
		if prev := heldAt(taken, key); prev != nil {
			if prev.res.Max != res.Max {
				return deny(fmt.Sprintf("%s: this token lists the same shared budget with two different limits, so the call was not sent", call.Tool), nil), nil
			}
			continue
		}
		out, err := g.store.Reserve(ctx, res)
		if err != nil {
			giveBack()
			return nil, fmt.Errorf("spend limit store: %w", err)
		}
		switch out.Status {
		case toolcontrols.OverLimit:
			left := leftUnits(l.MaxUnits(), out.Total)
			return deny(overMessage(call.Tool, l, sym, &left), &left), nil
		case toolcontrols.Duplicate:
			// The earlier call with this id holds a reservation. Do not
			// release it from here, and do not forward a second copy.
			giveBack()
			return &GateAdmission{Refusal: &GateRefusal{
				Code:    DenialDuplicateRequest,
				Message: fmt.Sprintf("%s: this request id was already used for the same call; send it with a new id to try again", call.Tool),
			}}, nil
		}
		taken = append(taken, held{res: res, limit: l, total: out.Total})
		if sym != "" {
			symbol = sym
		}
	}
	if len(taken) == 0 {
		return nil, nil // every limit was for another symbol
	}

	first := taken[0]
	detail := GateReceiptDetail{
		ArgumentField: first.limit.Field,
		Window:        string(first.limit.Window),
		Limit:         first.limit.Max,
		WindowTotal:   formatUnits(first.total),
		Symbol:        symbol,
	}
	for _, h := range taken {
		if h.limit.Budget != "" {
			detail.Shared = true
		}
	}
	return &GateAdmission{
		Detail: detail,
		Settle: func(ctx context.Context, o GateOutcome) {
			if o.Succeeded {
				return // the amount stays counted
			}
			if o.Unknown {
				// The call was sent and SatGate cannot tell whether it ran.
				// A lost reply is not a refusal: the order may have been
				// placed. Money that may have been spent stays counted.
				log.Warn().Str("tool", call.Tool).Int("limits", len(taken)).
					Msg("spend limit: outcome unknown; reservation kept")
				return
			}
			giveBack()
		},
	}, nil
}

func heldAt(taken []held, key string) *held {
	for i := range taken {
		if taken[i].res.Key == key {
			return &taken[i]
		}
	}
	return nil
}

// counterName names the counter a limit spends from, apart from tenant and
// window. A limit with no budget is named by its caveat word, so a token and
// the children that inherit its words share it. A limit with a budget is named
// by the budget, so every token minted into the budget shares it, whatever its
// words. A symbol cap inside a budget has a counter of its own per kind and
// symbol list, apart from the budget's total.
func counterName(l toolcontrols.SpendLimit) string {
	if l.Budget == "" {
		return l.ID
	}
	name := "b:" + l.Budget
	if len(l.Symbols) > 0 {
		name += ":s:" + l.SymbolKind() + ":" + strings.Join(l.Symbols, ",")
	}
	return name
}

func windowIDOf(l toolcontrols.SpendLimit, now time.Time) string {
	id, _, _ := l.WindowBounds(now)
	return id
}

// remainingAt reads what is left under a budget or symbol cap, for the
// refusal's data. It returns nil for any other limit (their refusals carry no
// remaining), and when the counter cannot be read (the refusal stands without it).
func (g *SpendGate) remainingAt(ctx context.Context, l toolcontrols.SpendLimit, key string) *int64 {
	if l.Budget == "" && len(l.Symbols) == 0 {
		return nil
	}
	total, err := g.store.Total(ctx, key)
	if err != nil {
		return nil
	}
	left := leftUnits(l.MaxUnits(), total)
	return &left
}

// leftUnits is max minus total, never below zero.
func leftUnits(max, total int64) int64 {
	if total >= max {
		return 0
	}
	return max - total
}

// logCounter names a counter in a log line without writing its key, which
// holds a budget id.
func logCounter(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12]
}

// sendFieldHint tells an agent that sent a quantity what to send instead.
func sendFieldHint(l toolcontrols.SpendLimit, why string) string {
	if why != argrules.NumericFieldMissing || !strings.HasPrefix(toolcontrols.FormatAmount(l.Field, "1"), "$") {
		return ""
	}
	return fmt.Sprintf("; send %s instead of quantity", l.Field)
}

// overMessage is the text for a call that would pass a limit. For a budget or a
// symbol cap it ends with what is left, in words and numbers only: never the
// budget's id or its owner's name.
func overMessage(tool string, l toolcontrols.SpendLimit, symbol string, left *int64) string {
	var who string
	switch {
	case l.Budget != "" && symbol != "":
		who = fmt.Sprintf("the limit for %s in the budget shared by your agents is %s", symbol, limitPhrase(l))
	case l.Budget != "":
		who = fmt.Sprintf("the limit shared by your agents is %s", limitPhrase(l))
	case symbol != "":
		who = fmt.Sprintf("this token's limit for %s is %s", symbol, limitPhrase(l))
	default:
		who = fmt.Sprintf("this token's limit is %s", limitPhrase(l))
	}
	msg := fmt.Sprintf("%s: %s on %s; this order would go over it", tool, who, l.Field)
	if left != nil && (l.Budget != "" || symbol != "") {
		msg += "; " + toolcontrols.FormatAmount(l.Field, formatUnits(*left)) + " is left " + l.Window.Today()
		switch {
		case symbol != "" && l.Budget != "":
			msg += " for " + symbol + " in the budget shared by your agents"
		case symbol != "":
			msg += " for " + symbol
		default:
			msg += " in the budget shared by your agents"
		}
	}
	return msg
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
		log.Error().Err(err).Str("counter", logCounter(r.Key)).Msg("could not release a spend reservation; the window total stays counted")
	}
}

func spendRefusal(tool string, l toolcontrols.SpendLimit, message string) *GateRefusal {
	return spendRefusalFor(l, "", message, nil)
}

// spendRefusalFor is the refusal for limit l. For a limit with a shared budget
// or a symbol cap the data and the receipt say so, with the canonical symbol
// and what is left when it was read. Never an id or a name.
func spendRefusalFor(l toolcontrols.SpendLimit, symbol, message string, remaining *int64) *GateRefusal {
	r := &GateRefusal{
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
	if l.Budget != "" {
		r.Data["shared"] = true
		r.Detail.Shared = true
	}
	if symbol != "" {
		r.Data["symbol"] = symbol
		r.Detail.Symbol = symbol
	}
	if remaining != nil {
		r.Data["remaining"] = formatUnits(*remaining)
		r.Detail.Remaining = formatUnits(*remaining)
	}
	return r
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
