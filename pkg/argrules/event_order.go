package argrules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
)

// Event contract orders: the most an order on a two-outcome market can cost,
// as one exact number.
//
// An event contract pays 1 when its outcome happens and 0 when it does not,
// so an order that opens exposure can cost at most contracts x price, and at
// most contracts x 1 when no price bounds it. The per-order cap (the
// "event_order" condition), the day and week limits and the "ask me" rules
// that carry "value":"event_max_cost", and the per-event stake limit all call
// EventOrderMaxCost. There is no second implementation: an order is valued, or
// refused for one fixed reason, the same way on every path. The function is
// pure (the same bytes in give the same answer out).
//
// Order tools differ from one service to the next, and a tool name does not
// say which service it belongs to, so nothing here reads a tool name. A token
// names a value schema instead, and the host program registers each schema it
// supports with RegisterEventSchema: the closed list of arguments the tool
// takes, and a reader that turns those arguments into an EventOrder (the
// market, the contract count, the price per contract the order can pay, and
// whether it can only close a position). This package does the arithmetic,
// the bounds and the refusals. A schema name that is not registered in the
// process refuses: a rule or control that names one does not parse, and a call
// valued under one is refused. This package registers no schema.
//
// What is never valued, whatever the schema: an argument outside the schema's
// list (including a letter-case twin of a listed one), a market id that is not
// 1 to 64 printable ASCII characters, a contract count that is not above zero
// with at most 2 decimals, a price that is not strictly between 0 and 1, and
// any shape the schema's reader refuses (an order that can rest past the day
// that counted it, an order sized by an amount, an amend or replace, a batch).
//
// Arithmetic is exact (big.Rat) and the result is rounded UP once to whole
// 10^-8 units, so a cost is never undercounted and never zero for an order
// that can open exposure. A cost of 10^10 or more is refused.

// EventCostBasis is the wire value of the "value" key on a spending limit or
// an "ask me" rule that counts event contract orders, and the meaning of the
// "event_order" condition and of an event stake limit.
const EventCostBasis = "event_max_cost"

// EventOrderCountedAs says in a few words what an event contract order is
// counted at. Every description of an event control uses it.
const EventOrderCountedAs = "contracts times the price per contract, or 1 per contract when the order has no price"

// EventOrderField is the reserved field name the event_order condition is
// written under. It is not a call argument.
const EventOrderField = "event_order"

// Bounds of a contract count. Counts are kept in hundredths of a contract.
const (
	MaxContractScale       = 2  // decimals in a contract count or a contracts cap
	MaxContractWholeDigits = 10 // digits before the point in a contracts cap
	ContractHundredths     = 100
)

// maxEventMarketLen bounds a market id.
const maxEventMarketLen = 64

// maxContractHundredths: a count of 10^MaxContractWholeDigits contracts or
// more is refused.
const maxContractHundredths int64 = 1000000000000

// Fixed reasons an event contract order cannot be valued. Like the other
// OrderCostWhy values they read after "this token counts each order at the
// most it can cost, and ", and none is ever built from the call.
const (
	WhyEventUnknownSchema    OrderCostWhy = "its value schema is not available here"
	WhyEventUnknownField     OrderCostWhy = "it carries an argument its value schema does not list"
	WhyEventNoMarket         OrderCostWhy = "it does not name its market as its value schema requires"
	WhyEventBadCount         OrderCostWhy = "its contract count is not a number above zero with at most 2 decimals"
	WhyEventTooManyContracts OrderCostWhy = "its contract count is too large to count"
	WhyEventBadPrice         OrderCostWhy = "its price is not a number between 0 and 1"
	WhyEventResting          OrderCostWhy = "an order that can stay open after the day it was placed (good till cancelled, or expiring later) cannot be counted"
	WhyEventByAmount         OrderCostWhy = "an order sized by an amount instead of a contract count cannot be counted"
	WhyEventAmendOrReplace   OrderCostWhy = "an order that amends or replaces another order cannot be counted"
	WhyEventBatch            OrderCostWhy = "a batch of orders cannot be counted"
	WhyEventClosingUnsure    OrderCostWhy = "it asks to only close a position in a way that cannot be guaranteed"
)

// eventWhys is every reason a schema's reader may return. Any other value is
// read as WhyUnreadable, so a reader cannot put text of its own (or a call
// value) in a refusal.
var eventWhys = map[OrderCostWhy]bool{
	OrderCostValued:          true,
	WhyUnreadable:            true,
	WhyBadNumber:             true,
	WhyNotAboveZero:          true,
	WhyEventNoMarket:         true,
	WhyEventBadCount:         true,
	WhyEventBadPrice:         true,
	WhyEventResting:          true,
	WhyEventByAmount:         true,
	WhyEventAmendOrReplace:   true,
	WhyEventBatch:            true,
	WhyEventClosingUnsure:    true,
	WhyEventTooManyContracts: true,
}

// EventOrder is what a schema's reader says an order is. The reader reads the
// call's own arguments and nothing else.
type EventOrder struct {
	// Market is the market id exactly as sent. It is handed to the host's
	// market lookup and never shown to the agent.
	Market string
	// Contracts is the number of contracts.
	Contracts *big.Rat
	// Price is the most the order can pay per contract for the exposure it
	// opens. nil means the order gives no price that bounds it; it is then
	// valued at 1 per contract.
	Price *big.Rat
	// Closing is true only when the service guarantees the order cannot open
	// exposure (it can only reduce a position the account holds). It is then
	// not counted against any cost limit; every other check still applies.
	Closing bool
}

// EventSchema is one registered value schema.
type EventSchema struct {
	// Name is the schema's name as tokens write it: 1 to 40 characters of
	// [a-z0-9_], starting with a letter. A released schema's meaning never
	// changes; a changed reading ships under a new name.
	Name string
	// Fields is the closed list of arguments the order tool takes. A call
	// with any other argument is refused, never valued.
	Fields []string
	// Read turns a call's arguments into an order, or returns one of the
	// fixed reasons above. It must be pure.
	Read func(args map[string]json.RawMessage) (EventOrder, OrderCostWhy)
}

var (
	eventSchemasMu sync.RWMutex
	eventSchemas   = map[string]*registeredSchema{}
)

type registeredSchema struct {
	fields map[string]bool
	read   func(args map[string]json.RawMessage) (EventOrder, OrderCostWhy)
}

// ErrEventSchema wraps every registration failure.
var ErrEventSchema = errors.New("event value schema")

// RegisterEventSchema makes a schema available to rules, controls and calls
// in this process. A name may be registered once; registering it again is an
// error, so a schema's meaning cannot change while tokens that name it are in
// use. The host program registers its schemas before it parses any token.
func RegisterEventSchema(s EventSchema) error {
	if !ValidEventSchemaName(s.Name) {
		return fmt.Errorf("%w: name %q is not 1 to 40 characters of [a-z0-9_] starting with a letter", ErrEventSchema, clip(s.Name))
	}
	if s.Read == nil {
		return fmt.Errorf("%w %s: no reader", ErrEventSchema, s.Name)
	}
	if len(s.Fields) == 0 || len(s.Fields) > 64 {
		return fmt.Errorf("%w %s: needs 1 to 64 fields", ErrEventSchema, s.Name)
	}
	fields := make(map[string]bool, len(s.Fields))
	for _, f := range s.Fields {
		if !fieldNameOK(f) || fields[f] {
			return fmt.Errorf("%w %s: field %q is not a field name or is listed twice", ErrEventSchema, s.Name, clip(f))
		}
		fields[f] = true
	}
	eventSchemasMu.Lock()
	defer eventSchemasMu.Unlock()
	if _, dup := eventSchemas[s.Name]; dup {
		return fmt.Errorf("%w %s: already registered", ErrEventSchema, s.Name)
	}
	eventSchemas[s.Name] = &registeredSchema{fields: fields, read: s.Read}
	return nil
}

// EventSchemaRegistered reports whether name is a registered schema.
func EventSchemaRegistered(name string) bool {
	eventSchemasMu.RLock()
	defer eventSchemasMu.RUnlock()
	_, ok := eventSchemas[name]
	return ok
}

// EventSchemaNames lists the registered schemas, sorted.
func EventSchemaNames() []string {
	eventSchemasMu.RLock()
	defer eventSchemasMu.RUnlock()
	out := make([]string, 0, len(eventSchemas))
	for n := range eventSchemas {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func lookupEventSchema(name string) *registeredSchema {
	eventSchemasMu.RLock()
	defer eventSchemasMu.RUnlock()
	return eventSchemas[name]
}

// ValidEventSchemaName reports whether name has the shape of a schema name.
func ValidEventSchemaName(name string) bool {
	if name == "" || len(name) > 40 || name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// EventOrderValue is an event contract order's most-it-can-cost value.
type EventOrderValue struct {
	// Units is the cost in whole 10^-8 units, rounded up; at least 1 for an
	// order that can open exposure, 0 for a Closing order.
	Units int64
	// Contracts is the contract count in hundredths of a contract (exact).
	Contracts int64
	// Market is the market id exactly as sent, for the host's lookup. It is
	// never put in a refusal, a receipt or a log.
	Market string
	// Closing: the order can only close a position. Nothing is counted for
	// it on any cost limit.
	Closing bool
}

// EventOrderMaxCostOfCall values a tools/call under schema. params is the
// exact params object the upstream will receive; sentTool must equal
// params.name. The call is read as strictly as argument rules read it (see
// CheckCallShape): a call that reads two ways is refused as unreadable.
func EventOrderMaxCostOfCall(params json.RawMessage, sentTool, schema string) (EventOrderValue, OrderCostWhy) {
	if CheckCallShape(params, sentTool) != NumericFieldOK {
		return EventOrderValue{}, WhyUnreadable
	}
	top, err := objectOf(params)
	if err != nil {
		return EventOrderValue{}, WhyUnreadable
	}
	var args map[string]json.RawMessage
	if raw, ok := top["arguments"]; ok {
		trimmed := bytes.TrimSpace(raw)
		if !bytes.Equal(trimmed, []byte("null")) {
			if args, err = objectOf(trimmed); err != nil {
				return EventOrderValue{}, WhyUnreadable
			}
		}
	}
	return EventOrderMaxCost(schema, args)
}

// EventOrderMaxCost values the arguments of an event contract order under a
// registered schema. args is params.arguments with unique keys (the callers
// have checked the call). The first failing step refuses:
//
//  1. The schema is registered.
//  2. Every argument is on the schema's list (exact name).
//  3. The schema's reader reads an order, or returns its fixed reason.
//  4. The market id is 1 to 64 printable ASCII characters, no space.
//  5. The contract count is above zero, has at most 2 decimals and is under
//     10^10.
//  6. A Closing order is valued at 0 and stops here.
//  7. The price, when given, is strictly between 0 and 1; no price means 1.
//  8. contracts x price rounds up to whole 10^-8 units; 10^10 or more is
//     refused.
func EventOrderMaxCost(schema string, args map[string]json.RawMessage) (v EventOrderValue, why OrderCostWhy) {
	s := lookupEventSchema(schema)
	if s == nil {
		return EventOrderValue{}, WhyEventUnknownSchema
	}
	for k := range args {
		if !s.fields[k] {
			return EventOrderValue{}, WhyEventUnknownField
		}
	}
	order, why := readEventOrder(s, args)
	if !eventWhys[why] {
		return EventOrderValue{}, WhyUnreadable
	}
	if !why.Valued() {
		return EventOrderValue{}, why
	}
	if !eventMarketOK(order.Market) {
		return EventOrderValue{}, WhyEventNoMarket
	}
	if order.Contracts == nil || order.Contracts.Sign() <= 0 {
		return EventOrderValue{}, WhyEventBadCount
	}
	hundredths := new(big.Rat).Mul(order.Contracts, big.NewRat(ContractHundredths, 1))
	if !hundredths.IsInt() {
		return EventOrderValue{}, WhyEventBadCount
	}
	if !hundredths.Num().IsInt64() || hundredths.Num().Int64() >= maxContractHundredths {
		return EventOrderValue{}, WhyEventTooManyContracts
	}
	out := EventOrderValue{Contracts: hundredths.Num().Int64(), Market: order.Market}
	if order.Closing {
		out.Closing = true
		return out, OrderCostValued
	}
	price := big.NewRat(1, 1)
	if order.Price != nil {
		if order.Price.Sign() <= 0 || order.Price.Cmp(big.NewRat(1, 1)) >= 0 {
			return EventOrderValue{}, WhyEventBadPrice
		}
		price = order.Price
	}
	cost, cwhy := costOf(new(big.Rat).Mul(order.Contracts, price))
	if !cwhy.Valued() {
		return EventOrderValue{}, cwhy
	}
	out.Units = cost.Units
	return out, OrderCostValued
}

// readEventOrder runs a schema's reader. A reader that panics refuses the
// call instead of taking the gateway down.
func readEventOrder(s *registeredSchema, args map[string]json.RawMessage) (order EventOrder, why OrderCostWhy) {
	defer func() {
		if recover() != nil {
			order, why = EventOrder{}, WhyUnreadable
		}
	}()
	if args == nil {
		args = map[string]json.RawMessage{}
	}
	return s.read(args)
}

func eventMarketOK(m string) bool {
	if m == "" || len(m) > maxEventMarketLen {
		return false
	}
	for i := 0; i < len(m); i++ {
		if m[i] <= 0x20 || m[i] >= 0x7f {
			return false
		}
	}
	return true
}

// Helpers for schema readers. Each reads one argument the way every other
// path in this package reads it.

// ReadArgString reads a JSON string argument. Anything else is not text.
func ReadArgString(raw json.RawMessage) (string, bool) {
	return jsonString(bytes.TrimSpace(raw))
}

// ReadArgDecimal reads a JSON number by its literal text, or a JSON string
// holding a plain decimal. No float64 is involved.
func ReadArgDecimal(raw json.RawMessage) (*big.Rat, bool) {
	d, ok := argumentDecimal(bytes.TrimSpace(raw))
	if !ok {
		return nil, false
	}
	return d.Rat(), true
}

// ReadArgBool reads a JSON true or false.
func ReadArgBool(raw json.RawMessage) (bool, bool) {
	switch string(bytes.TrimSpace(raw)) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

// FormatContracts writes hundredths of a contract as a plain decimal.
func FormatContracts(hundredths int64) string {
	whole, frac := hundredths/ContractHundredths, hundredths%ContractHundredths
	if frac == 0 {
		return fmt.Sprintf("%d", whole)
	}
	s := fmt.Sprintf("%d.%02d", whole, frac)
	if s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	return s
}

// EventOrderLimit is the per-order cap on an event contract order: the
// condition written under the reserved field "event_order". Schema is the
// value schema; MaxCost caps the order's value (EventOrderMaxCost) and
// MaxContracts its contract count. At least one of the two is set.
type EventOrderLimit struct {
	Schema       string
	MaxCost      *Decimal
	MaxContracts *Decimal
}

// eventOrderHolds is the per-order cap for one event_order condition: the
// call is valued, and passes when its cost is at most MaxCost (a Closing
// order always passes the cost cap) and its count at most MaxContracts.
func (c Condition) eventOrderHolds(args map[string]json.RawMessage) (bool, OrderCostWhy) {
	v, why := EventOrderMaxCost(c.EventOrder.Schema, args)
	if !why.Valued() {
		return false, why
	}
	if c.EventOrder.MaxCost != nil && !v.Closing {
		capUnits := new(big.Rat).Mul(c.EventOrder.MaxCost.rat, new(big.Rat).SetInt64(UnitsPerOne))
		if new(big.Rat).SetInt64(v.Units).Cmp(capUnits) > 0 {
			return false, why
		}
	}
	if c.EventOrder.MaxContracts != nil {
		capHundredths := new(big.Rat).Mul(c.EventOrder.MaxContracts.rat, new(big.Rat).SetInt64(ContractHundredths))
		if new(big.Rat).SetInt64(v.Contracts).Cmp(capHundredths) > 0 {
			return false, why
		}
	}
	return true, why
}

// parseEventOrderCondition reads {"schema":..., "max_cost":..., "max_contracts":...}
// under the reserved field event_order.
func parseEventOrderCondition(tool string, obj map[string]json.RawMessage) (Condition, error) {
	lim := &EventOrderLimit{}
	sawSchema := false
	for k, v := range obj {
		switch k {
		case "schema":
			if err := json.Unmarshal(v, &lim.Schema); err != nil {
				return Condition{}, ruleErr("schema for %s.%s must be a string", tool, EventOrderField)
			}
			sawSchema = true
		case "max_cost":
			d, err := parseConditionDecimal(v)
			if err != nil {
				return Condition{}, ruleErr("max_cost for %s.%s is not a plain decimal", tool, EventOrderField)
			}
			lim.MaxCost = d
		case "max_contracts":
			d, err := parseConditionDecimal(v)
			if err != nil {
				return Condition{}, ruleErr("max_contracts for %s.%s is not a plain decimal", tool, EventOrderField)
			}
			lim.MaxContracts = d
		default:
			return Condition{}, ruleErr("unknown condition %q for %s.%s", clip(k), tool, EventOrderField)
		}
	}
	if !sawSchema {
		return Condition{}, ruleErr("%s.%s needs a schema", tool, EventOrderField)
	}
	if !EventSchemaRegistered(lim.Schema) {
		return Condition{}, ruleErr("%s.%s names value schema %q, which is not registered here", tool, EventOrderField, clip(lim.Schema))
	}
	if lim.MaxCost == nil && lim.MaxContracts == nil {
		return Condition{}, ruleErr("%s.%s needs max_cost or max_contracts", tool, EventOrderField)
	}
	if lim.MaxCost != nil {
		if err := boundedPositive(lim.MaxCost, MaxOrderCostWholeDigits, MaxOrderCostScale); err != nil {
			return Condition{}, ruleErr("max_cost for %s.%s %v", tool, EventOrderField, err)
		}
	}
	if lim.MaxContracts != nil {
		if err := boundedPositive(lim.MaxContracts, MaxContractWholeDigits, MaxContractScale); err != nil {
			return Condition{}, ruleErr("max_contracts for %s.%s %v", tool, EventOrderField, err)
		}
	}
	return Condition{Field: EventOrderField, EventOrder: lim}, nil
}

// boundedPositive checks a decimal is above zero with at most whole digits
// before and scale digits after the point.
func boundedPositive(d *Decimal, whole, scale int) error {
	if d.rat.Sign() <= 0 {
		return errors.New("must be above zero")
	}
	w, f, _ := cutPoint(d.String())
	if len(w) > whole || len(f) > scale {
		return fmt.Errorf("has more than %d digits before or %d after the point", whole, scale)
	}
	return nil
}

func cutPoint(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

// clauses is "costs at most 25 and is at most 100 contracts".
func (l *EventOrderLimit) clauses() string {
	var parts []string
	if l.MaxCost != nil {
		parts = append(parts, "costs at most "+l.MaxCost.String())
	}
	if l.MaxContracts != nil {
		parts = append(parts, "is at most "+l.MaxContracts.String()+" contracts")
	}
	return joinAnd(parts)
}

// agentMust is "cost at most 25 and be at most 100 contracts".
func (l *EventOrderLimit) agentMust() string {
	var parts []string
	if l.MaxCost != nil {
		parts = append(parts, "cost at most "+l.MaxCost.String())
	}
	if l.MaxContracts != nil {
		parts = append(parts, "be at most "+l.MaxContracts.String()+" contracts")
	}
	return joinAnd(parts)
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " and "
		}
		out += p
	}
	return out
}
