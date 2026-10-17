package argrules

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
)

// Order cost: the most a place_equity_order or place_crypto_order call can
// cost, as one exact number.
//
// The per-order cap (the "order" condition with max_cost), the day and week
// spending limits that carry "value":"order_max_cost", the shared-budget
// limits that carry it, and the "ask me" rules that carry it all call
// OrderMaxCost. There is no second implementation: a call is valued, or
// refused for one fixed reason, the same way on every path. The function is
// pure (the same bytes in give the same answer out), so it may run once per
// path for one call.
//
// What a call is valued at:
//
//   - By dollars (dollar_amount present, quantity absent): dollar_amount.
//     limit_price or stop_price beside it is refused. side and type are not
//     read, so a sell by dollars is valued as it always was.
//   - By shares (quantity present): quantity x limit_price, and only for a
//     day limit buy of shares: tool place_equity_order, side buy, type limit
//     or stop_limit, time_in_force gfd, market_hours absent, regular_hours or
//     extended_hours. Every other call that gives quantity is refused. An
//     order that can stay open past the day or week that counted it could fill
//     in a later window that never counted it, which would undercount, so no
//     good-till-canceled order is valued, and neither is an order in the
//     overnight session (it can fill after midnight).
//
// A limit buy is counted at the most it can cost when it is placed. An order
// that does not fill still counts until the day or week resets.
//
// Arithmetic is exact (big.Rat) and the result is rounded UP to a whole number
// of 10^-8 units, so a cost is never undercounted and never zero. Nothing is
// rounded to cents. A cost of 10^10 or more is refused: no cap, limit or
// threshold can state it (see MaxOrderCostWholeDigits).

// OrderCostBasis is the wire value of the "value" key on a spending limit or
// an "ask me" rule, and the meaning of the "max_cost" condition: count each
// order at the most it can cost.
const OrderCostBasis = "order_max_cost"

// OrderCostCountedAs says in a few words what an order is counted at. Every
// agent-facing description of an order-cost control uses it, so the words are
// written once.
const OrderCostCountedAs = "dollar_amount, or quantity times limit_price for a day limit buy of shares"

// The size bounds of a cap, limit or threshold written for order cost. They
// are the bounds toolcontrols.MaxWholeDigits and toolcontrols.MaxScale give
// a spending limit; a test holds the two pairs equal (toolcontrols imports
// this package, so the numbers cannot be shared).
const (
	MaxOrderCostWholeDigits = 10
	MaxOrderCostScale       = 8
)

// maxOrderCostUnits is 10^MaxOrderCostWholeDigits in 10^-8 units. A cost
// that rounds to this or more is refused.
const maxOrderCostUnits int64 = 1000000000000000000

// The allow-lists below decide which calls the shares path may value. Each is
// one named constant so that a later owner decision to widen one is a
// one-line change here (and its test), with no change to the wire format or
// to any other file. String matches are ASCII case-insensitive; a value with
// a non-ASCII byte never matches.
//
// orderCostTools: the tools OrderMaxCost values at all (the dollars path, and
// the tools a limit, rule or cap may name).
var orderCostTools = [...]string{"place_equity_order", "place_crypto_order"}

// orderCostSharesTools: the tools that may use the shares path (quantity).
// place_crypto_order is not here: its limit orders are always good till
// canceled for 90 days, so they cannot be counted yet.
var orderCostSharesTools = [...]string{"place_equity_order"}

// orderCostSharesTypes: the order types the shares path values. A market
// order by shares has no price to multiply; stop and stop_market orders have
// no limit_price that bounds their cost.
var orderCostSharesTypes = [...]string{"limit", "stop_limit"}

// orderCostSharesTimeInForce: the time_in_force values the shares path values,
// and the field must be present. gfd is a day order.
var orderCostSharesTimeInForce = [...]string{"gfd"}

// orderCostSharesMarketHours: the market_hours values the shares path values,
// when the field is present. all_day_hours is the overnight session, where an
// order can fill after midnight.
var orderCostSharesMarketHours = [...]string{"regular_hours", "extended_hours"}

// orderCostSharesSides: the sides the shares path values.
var orderCostSharesSides = [...]string{"buy"}

// OrderCost is a call's most-it-can-cost value.
type OrderCost struct {
	// Units is the cost in whole 10^-8 units, rounded up, always at least 1.
	Units int64
}

// OrderCostWhy says why a call could not be valued. The empty value means it
// was valued. Every other value is one of the fixed phrases below, written to
// read after "this token counts each order at the most it can cost, and ".
// None is ever built from the call, so none can carry a call's value.
type OrderCostWhy string

const (
	// OrderCostValued: the call was valued.
	OrderCostValued OrderCostWhy = ""

	WhyUnreadable       OrderCostWhy = "it could not be read"
	WhyNotOrderTool     OrderCostWhy = "only place_equity_order and place_crypto_order can be counted this way"
	WhyBothSizes        OrderCostWhy = "it gives both dollar_amount and quantity"
	WhyNoSize           OrderCostWhy = "it gives neither dollar_amount nor quantity"
	WhyDollarsWithPrice OrderCostWhy = "an order by dollar_amount cannot also give limit_price or stop_price"
	WhySharesToolNotYet OrderCostWhy = "crypto orders by quantity can't be counted yet"
	WhyNotABuy          OrderCostWhy = "an order by quantity is only counted when side is buy"
	WhyNotALimitBuy     OrderCostWhy = "an order by quantity is only counted for a limit or stop-limit buy; market and stop orders by quantity cannot be counted"
	WhyNoTimeInForce    OrderCostWhy = "an order by quantity must give time_in_force gfd (a day order)"
	WhyNotADayOrder     OrderCostWhy = "an order by quantity is only counted as a day order (time_in_force gfd), because an order that stays open can fill after the day or week that counted it"
	WhyOvernightSession OrderCostWhy = "an order by quantity is only counted in regular or extended hours, because an overnight order can fill after midnight"
	WhyNoLimitPrice     OrderCostWhy = "a limit buy by quantity must give limit_price"
	WhyBadNumber        OrderCostWhy = "a number in it is not a plain number"
	WhyNotAboveZero     OrderCostWhy = "a number in it is not above zero"
	WhyCostTooLarge     OrderCostWhy = "its cost is too large to count"
)

// Valued reports whether the call was valued.
func (w OrderCostWhy) Valued() bool { return w == OrderCostValued }

// orderCostFields are the top-level arguments OrderMaxCost reads. A call
// argument whose name differs from one of them only by letter case is not read
// at all and the call is refused as unreadable: an upstream that matched names
// without regard to case could read it as the real field.
var orderCostFields = [...]string{
	"dollar_amount", "quantity", "limit_price", "stop_price",
	"side", "type", "time_in_force", "market_hours",
}

// IsOrderTool reports whether tool names an order tool OrderMaxCost values,
// without regard to ASCII letter case.
func IsOrderTool(tool string) bool { return inFoldedList(orderCostTools[:], tool) }

// inFoldedList reports whether v equals a list entry, ASCII case-insensitively.
// A value with a non-ASCII byte never matches.
func inFoldedList(list []string, v string) bool {
	for i := 0; i < len(v); i++ {
		if v[i] >= 0x80 {
			return false
		}
	}
	lv := asciiLower(v)
	for _, s := range list {
		if s == lv {
			return true
		}
	}
	return false
}

// OrderMaxCostOfCall values a tools/call. params is the exact params object
// the upstream will receive; sentTool is the name the call was sent under (it
// must equal params.name, as CheckCallShape requires); orderTool is the order
// tool the call stands for. They differ only for a tool listed under a renamed
// name, whose own name a shared budget is written for. The call is read as
// strictly as argument rules read it (see CheckCallShape): a call that reads
// two ways is refused as unreadable, not valued.
func OrderMaxCostOfCall(params json.RawMessage, sentTool, orderTool string) (OrderCost, OrderCostWhy) {
	if CheckCallShape(params, sentTool) != NumericFieldOK {
		return OrderCost{}, WhyUnreadable
	}
	top, err := objectOf(params)
	if err != nil {
		return OrderCost{}, WhyUnreadable
	}
	var args map[string]json.RawMessage
	if raw, ok := top["arguments"]; ok {
		trimmed := bytes.TrimSpace(raw)
		if !bytes.Equal(trimmed, []byte("null")) {
			if args, err = objectOf(trimmed); err != nil {
				return OrderCost{}, WhyUnreadable
			}
		}
	}
	return OrderMaxCost(orderTool, args)
}

// OrderMaxCost values the arguments of a call to tool. args is
// params.arguments, already decoded as an object whose keys are unique (the
// callers have checked the call with CheckCallShape or Check). The first
// failing step refuses; see the comment at the top of this file.
//
//  1. tool is place_equity_order or place_crypto_order.
//  2. No argument name differs from a field it reads only by letter case.
//  3. dollar_amount and quantity are not both present, and one is. A key that
//     is present with null counts as present.
//  4. By dollars: limit_price and stop_price absent; dollar_amount a plain
//     decimal above zero.
//  5. By shares: the tool, side, type, time_in_force and market_hours pass
//     their allow-lists; limit_price present, a plain decimal above zero;
//     quantity a plain decimal above zero. stop_price is not read (it cannot
//     raise a stop-limit's cost above quantity x limit_price).
//  6. The exact value rounds up to whole 10^-8 units; a cost of 10^10 or more
//     is refused.
func OrderMaxCost(tool string, args map[string]json.RawMessage) (OrderCost, OrderCostWhy) {
	if !IsOrderTool(tool) {
		return OrderCost{}, WhyNotOrderTool
	}
	for k := range args {
		for _, f := range orderCostFields {
			if k != f && strings.EqualFold(k, f) {
				return OrderCost{}, WhyUnreadable
			}
		}
	}
	dollarsRaw, hasDollars := args["dollar_amount"]
	quantityRaw, hasQuantity := args["quantity"]
	switch {
	case hasDollars && hasQuantity:
		return OrderCost{}, WhyBothSizes
	case !hasDollars && !hasQuantity:
		return OrderCost{}, WhyNoSize
	}

	if hasDollars {
		_, hasLimit := args["limit_price"]
		_, hasStop := args["stop_price"]
		if hasLimit || hasStop {
			return OrderCost{}, WhyDollarsWithPrice
		}
		d, ok := argumentDecimal(bytes.TrimSpace(dollarsRaw))
		if !ok {
			return OrderCost{}, WhyBadNumber
		}
		return costOf(d.rat)
	}

	// By shares.
	if !inFoldedList(orderCostSharesTools[:], tool) {
		return OrderCost{}, WhySharesToolNotYet
	}
	if !argIn(args, "side", orderCostSharesSides[:]) {
		return OrderCost{}, WhyNotABuy
	}
	if !argIn(args, "type", orderCostSharesTypes[:]) {
		return OrderCost{}, WhyNotALimitBuy
	}
	if _, present := args["time_in_force"]; !present {
		return OrderCost{}, WhyNoTimeInForce
	}
	if !argIn(args, "time_in_force", orderCostSharesTimeInForce[:]) {
		return OrderCost{}, WhyNotADayOrder
	}
	if _, present := args["market_hours"]; present && !argIn(args, "market_hours", orderCostSharesMarketHours[:]) {
		return OrderCost{}, WhyOvernightSession
	}
	limitRaw, hasLimit := args["limit_price"]
	if !hasLimit {
		return OrderCost{}, WhyNoLimitPrice
	}
	price, ok := argumentDecimal(bytes.TrimSpace(limitRaw))
	if !ok {
		return OrderCost{}, WhyBadNumber
	}
	qty, ok := argumentDecimal(bytes.TrimSpace(quantityRaw))
	if !ok {
		return OrderCost{}, WhyBadNumber
	}
	if price.rat.Sign() <= 0 || qty.rat.Sign() <= 0 {
		return OrderCost{}, WhyNotAboveZero
	}
	return costOf(qty.Times(price))
}

// argIn reports whether args[name] is a JSON string equal, ASCII
// case-insensitively, to a list entry. A missing key, null or any other JSON
// type is not.
func argIn(args map[string]json.RawMessage, name string, list []string) bool {
	raw, present := args[name]
	if !present {
		return false
	}
	s, ok := jsonString(bytes.TrimSpace(raw))
	return ok && inFoldedList(list, s)
}

// costOf rounds an exact value up to units and applies the bounds.
func costOf(r *big.Rat) (OrderCost, OrderCostWhy) {
	if r.Sign() <= 0 {
		return OrderCost{}, WhyNotAboveZero
	}
	units, ok := UnitsCeil(r)
	if !ok || units >= maxOrderCostUnits {
		return OrderCost{}, WhyCostTooLarge
	}
	// A value above zero rounds up to at least 1 unit; this keeps "never
	// zero" true even if UnitsCeil were ever changed.
	if units < 1 {
		units = 1
	}
	return OrderCost{Units: units}, OrderCostValued
}

// orderCostHolds is the per-order cap for one condition of rule: the call is
// valued, and passes when its cost is at most the cap. A call that cannot be
// valued does not pass. The reason is returned for the refusal's wording.
func (c Condition) orderCostHolds(tool string, args map[string]json.RawMessage) (bool, OrderCostWhy) {
	cost, why := OrderMaxCost(tool, args)
	if !why.Valued() {
		return false, why
	}
	// Compare exactly against the cap in units. A cap is written with at most
	// MaxOrderCostScale fractional digits, so this is exact; a cap built by
	// hand with more is still compared exactly, not rounded in its favour.
	capUnits := new(big.Rat).Mul(c.MaxCost.rat, new(big.Rat).SetInt64(UnitsPerOne))
	return new(big.Rat).SetInt64(cost.Units).Cmp(capUnits) <= 0, why
}
