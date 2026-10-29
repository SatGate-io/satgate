package toolcontrols

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// Event contract controls (opt-in).
//
// An event contract order is valued by argrules.EventOrderMaxCost under a
// value schema the host program registered (see argrules.RegisterEventSchema).
// Four controls use that one value:
//
//   - a spending limit or an "ask me" rule with "value":"event_max_cost" and
//     "schema":"<name>" counts or compares each order at the most it can cost;
//   - "event_markets" allows only markets whose series or category is listed;
//   - "event_stakes" caps the total counted on any one event.
//
// (The per-order cap and contracts cap are argument rules: the "event_order"
// condition in pkg/argrules.)
//
//	{"v":1,
//	 "spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"50","window":"day",
//	                  "tz":"America/New_York","value":"event_max_cost","schema":"<name>"}],
//	 "approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"20",
//	               "value":"event_max_cost","schema":"<name>"}],
//	 "event_markets":[{"tools":["place_event_order"],"schema":"<name>",
//	                   "categories":["ECONOMICS"],"series":["<series id>"]}],
//	 "event_stakes":[{"tools":["place_event_order"],"schema":"<name>","max":"40","reset":"manual"}]}
//
// "schema" must name a schema registered in the process that reads the
// document; one that is not refuses the document. The market's event, series
// and category are not in the order: the gateway asks a market lookup the
// host program registered for the schema (see mcpserver.EventGate), and a
// call it cannot look up is refused.
//
// An event_markets entry passes a call when the market's series is listed or
// its category is listed (each list optional, at least one given, at most
// MaxEventListEntries each). Categories are written trimmed and in ASCII upper
// case, and a looked-up category is compared after the same folding. Several
// entries must all pass, so a delegated token can only narrow. There is no
// block list: a category that appears later is simply not on an allow list.
//
// An event_stakes entry caps the sum of the values of the orders counted on
// one event (every market of the event together). "reset" is "manual": the
// total keeps adding up until the host program resets it (EventStakeStore
// ResetEvents). An order that can only close a position is not counted.
//
// A release that predates these keys refuses the document (unknown key):
// it fails closed. Marshal writes them only when set, so every existing
// document keeps its canonical bytes.

// DenialCodeEventMarket is the stable code of a call refused because its
// market is not one the token allows, or could not be confirmed as a
// two-outcome market worth at most 1 per contract that is open for trading.
const DenialCodeEventMarket = "EVENT_MARKET_NOT_ALLOWED"

// Size limits of the event controls.
const (
	MaxEventMarketRules = 8  // event_markets entries over all caveats
	MaxEventStakes      = 8  // event_stakes entries over all caveats
	MaxEventListEntries = 32 // categories, and series, in one entry
	maxEventListEntry   = 64 // bytes in one category or series
)

// EventResetManual is the only reset of an event stake in this version.
const EventResetManual = "manual"

// EventMarketRule allows only some markets of an event contract tool.
type EventMarketRule struct {
	// Tools are the tools the rule covers, sorted.
	Tools []string
	// Schema is the value schema whose market lookup answers for the call.
	Schema string
	// Categories are allowed categories, trimmed and ASCII upper case, sorted.
	Categories []string
	// Series are allowed series ids, compared exactly, sorted.
	Series []string
}

// Covers reports whether the rule applies to tool (letter case ignored, like
// SpendLimit.Covers).
func (r EventMarketRule) Covers(tool string) bool { return coversTool(r.Tools, tool) }

// Allows reports whether a market with this series and category passes.
func (r EventMarketRule) Allows(series, category string) bool {
	if series != "" {
		for _, s := range r.Series {
			if s == series {
				return true
			}
		}
	}
	if c := CanonicalEventCategory(category); c != "" {
		for _, x := range r.Categories {
			if x == c {
				return true
			}
		}
	}
	return false
}

// CanonicalEventCategory trims ASCII white space and upper-cases ASCII
// letters. Text with a byte outside printable ASCII has no canonical form
// ("") and matches nothing.
func CanonicalEventCategory(s string) string {
	s = asciiTrim(s)
	b := []byte(s)
	for i, c := range b {
		if c < 0x20 || c >= 0x7f {
			return ""
		}
		if c >= 'a' && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}

// EventStakeLimit caps the total counted on any one event.
type EventStakeLimit struct {
	// Tools are the tools the limit covers, sorted.
	Tools []string
	// Schema is the value schema the orders are valued under.
	Schema string
	// Max is the most counted on one event, as written.
	Max string
	// Reset is EventResetManual.
	Reset string

	// ID names the limit's counter (set by ParseScopeValue). A limit with no
	// ID cannot be enforced.
	ID string

	units int64
}

// Covers reports whether the limit applies to tool (letter case ignored).
func (l EventStakeLimit) Covers(tool string) bool { return coversTool(l.Tools, tool) }

// MaxUnits is Max in whole 10^-8 units.
func (l EventStakeLimit) MaxUnits() int64 { return l.units }

func coversTool(tools []string, tool string) bool {
	for _, t := range tools {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

// validateTools checks a tool list and sorts it.
func validateTools(tools []string, what string) error {
	if len(tools) == 0 || len(tools) > MaxToolsPerLimit {
		return ctlErr("%s covers 1 to %d tools", what, MaxToolsPerLimit)
	}
	seen := map[string]bool{}
	for _, t := range tools {
		if !argrules.ValidToolName(t) {
			return ctlErr("tool name is not allowed")
		}
		if seen[t] {
			return ctlErr("a tool is listed twice")
		}
		seen[t] = true
	}
	sort.Strings(tools)
	return nil
}

// validateSchema checks that a schema name is registered.
func validateSchema(schema string) error {
	if schema == "" {
		return ctlErr("schema must name a value schema")
	}
	if !argrules.EventSchemaRegistered(schema) {
		return ctlErr("schema %q is not registered here", clip(schema))
	}
	return nil
}

func parseEventMarketRule(raw json.RawMessage) (EventMarketRule, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return EventMarketRule{}, ctlErr("an event_markets entry must be an object")
	}
	var r EventMarketRule
	var sawTools, sawSchema bool
	for k, v := range obj {
		switch k {
		case "tools":
			if err := json.Unmarshal(v, &r.Tools); err != nil {
				return EventMarketRule{}, ctlErr("tools must be a list of names")
			}
			sawTools = true
		case "schema":
			if err := json.Unmarshal(v, &r.Schema); err != nil {
				return EventMarketRule{}, ctlErr("schema must be a string")
			}
			sawSchema = true
		case "categories":
			if err := json.Unmarshal(v, &r.Categories); err != nil || r.Categories == nil {
				return EventMarketRule{}, ctlErr("categories must be a list of strings")
			}
		case "series":
			if err := json.Unmarshal(v, &r.Series); err != nil || r.Series == nil {
				return EventMarketRule{}, ctlErr("series must be a list of strings")
			}
		default:
			return EventMarketRule{}, ctlErr("unknown event_markets key %q", clip(k))
		}
	}
	if !sawTools || !sawSchema {
		return EventMarketRule{}, ctlErr("an event_markets entry needs tools and schema")
	}
	if err := r.validate(); err != nil {
		return EventMarketRule{}, err
	}
	return r, nil
}

func (r *EventMarketRule) validate() error {
	if err := validateTools(r.Tools, "an event_markets entry"); err != nil {
		return err
	}
	if err := validateSchema(r.Schema); err != nil {
		return err
	}
	if len(r.Categories) == 0 && len(r.Series) == 0 {
		return ctlErr("an event_markets entry needs categories or series")
	}
	if r.Categories != nil {
		if err := validateEventList(r.Categories, "categories", true); err != nil {
			return err
		}
	}
	if r.Series != nil {
		if err := validateEventList(r.Series, "series", false); err != nil {
			return err
		}
	}
	return nil
}

// validateEventList checks a categories or series list: 1 to
// MaxEventListEntries entries of 1 to 64 printable ASCII characters, no
// repeat; categories already in canonical form. It sorts the list.
func validateEventList(list []string, name string, category bool) error {
	if len(list) == 0 || len(list) > MaxEventListEntries {
		return ctlErr("%s lists 1 to %d entries", name, MaxEventListEntries)
	}
	seen := map[string]bool{}
	for _, s := range list {
		if s == "" || len(s) > maxEventListEntry {
			return ctlErr("an entry of %s is empty or longer than %d bytes", name, maxEventListEntry)
		}
		for i := 0; i < len(s); i++ {
			if s[i] < 0x20 || s[i] >= 0x7f {
				return ctlErr("an entry of %s is not printable ASCII", name)
			}
		}
		if category && CanonicalEventCategory(s) != s {
			return ctlErr("category %q is not written trimmed and in upper case", clip(s))
		}
		if !category && asciiTrim(s) != s {
			return ctlErr("a series id has white space at an end")
		}
		if seen[s] {
			return ctlErr("an entry of %s is listed twice", name)
		}
		seen[s] = true
	}
	sort.Strings(list)
	return nil
}

func parseEventStakeLimit(raw json.RawMessage) (EventStakeLimit, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return EventStakeLimit{}, ctlErr("an event_stakes entry must be an object")
	}
	var l EventStakeLimit
	var sawTools, sawSchema, sawMax, sawReset bool
	for k, v := range obj {
		switch k {
		case "tools":
			if err := json.Unmarshal(v, &l.Tools); err != nil {
				return EventStakeLimit{}, ctlErr("tools must be a list of names")
			}
			sawTools = true
		case "schema":
			if err := json.Unmarshal(v, &l.Schema); err != nil {
				return EventStakeLimit{}, ctlErr("schema must be a string")
			}
			sawSchema = true
		case "max":
			if err := json.Unmarshal(v, &l.Max); err != nil {
				return EventStakeLimit{}, ctlErr("max must be a string holding a plain decimal")
			}
			sawMax = true
		case "reset":
			if err := json.Unmarshal(v, &l.Reset); err != nil {
				return EventStakeLimit{}, ctlErr("reset must be a string")
			}
			sawReset = true
		default:
			return EventStakeLimit{}, ctlErr("unknown event_stakes key %q", clip(k))
		}
	}
	if !sawTools || !sawSchema || !sawMax || !sawReset {
		return EventStakeLimit{}, ctlErr("an event_stakes entry needs tools, schema, max and reset")
	}
	if err := l.validate(); err != nil {
		return EventStakeLimit{}, err
	}
	return l, nil
}

func (l *EventStakeLimit) validate() error {
	if err := validateTools(l.Tools, "an event_stakes entry"); err != nil {
		return err
	}
	if err := validateSchema(l.Schema); err != nil {
		return err
	}
	if l.Reset != EventResetManual {
		return ctlErr("reset must be %q", EventResetManual)
	}
	units, err := maxUnitsOf(l.Max)
	if err != nil {
		return err
	}
	l.units = units
	return nil
}

// maxUnitsOf reads a limit's max: a plain decimal above zero with at most
// MaxWholeDigits before and MaxScale after the point.
func maxUnitsOf(max string) (int64, error) {
	d, err := argrules.ParseDecimal(max)
	if err != nil {
		return 0, ctlErr("max is not a plain decimal")
	}
	rat := d.Rat()
	if rat.Sign() <= 0 {
		return 0, ctlErr("max must be above zero")
	}
	whole, frac, _ := strings.Cut(max, ".")
	if len(whole) > MaxWholeDigits || len(frac) > MaxScale {
		return 0, ctlErr("max has more than %d digits before or %d after the point", MaxWholeDigits, MaxScale)
	}
	units, ok := unitsOf(rat, false)
	if !ok || units <= 0 {
		return 0, ctlErr("max is out of range")
	}
	return units, nil
}

func parseEventList[T any](raw json.RawMessage, name string, max int, parse func(json.RawMessage) (T, error)) ([]T, error) {
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, ctlErr("%s must be a list", name)
	}
	if len(list) == 0 {
		return nil, ctlErr("%s is empty", name)
	}
	if len(list) > max {
		return nil, ctlErr("more than %d %s entries", max, name)
	}
	out := make([]T, 0, len(list))
	for _, r := range list {
		v, err := parse(r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// marshalEventControls appends the event_markets and event_stakes keys.
func marshalEventControls(b *strings.Builder, d *Doc) error {
	if len(d.EventMarkets) > 0 {
		b.WriteString(`,"event_markets":[`)
		for i, r := range d.EventMarkets {
			cp := r
			cp.Tools = append([]string(nil), r.Tools...)
			if r.Categories != nil {
				cp.Categories = append([]string(nil), r.Categories...)
			}
			if r.Series != nil {
				cp.Series = append([]string(nil), r.Series...)
			}
			if err := cp.validate(); err != nil {
				return err
			}
			if i > 0 {
				b.WriteByte(',')
			}
			tools, _ := json.Marshal(cp.Tools)
			schema, _ := json.Marshal(cp.Schema)
			fmt.Fprintf(b, `{"tools":%s,"schema":%s`, tools, schema)
			if cp.Categories != nil {
				cj, _ := json.Marshal(cp.Categories)
				fmt.Fprintf(b, `,"categories":%s`, cj)
			}
			if cp.Series != nil {
				sj, _ := json.Marshal(cp.Series)
				fmt.Fprintf(b, `,"series":%s`, sj)
			}
			b.WriteByte('}')
		}
		b.WriteString(`]`)
	}
	if len(d.EventStakes) > 0 {
		b.WriteString(`,"event_stakes":[`)
		for i, l := range d.EventStakes {
			cp := l
			cp.Tools = append([]string(nil), l.Tools...)
			if err := cp.validate(); err != nil {
				return err
			}
			if i > 0 {
				b.WriteByte(',')
			}
			tools, _ := json.Marshal(cp.Tools)
			schema, _ := json.Marshal(cp.Schema)
			max, _ := json.Marshal(cp.Max)
			reset, _ := json.Marshal(cp.Reset)
			fmt.Fprintf(b, `{"tools":%s,"schema":%s,"max":%s,"reset":%s}`, tools, schema, max, reset)
		}
		b.WriteString(`]`)
	}
	return nil
}

// CollectEventMarkets reads every control caveat on a token, in order, and
// returns its event_markets entries.
func CollectEventMarkets(caveats []string) ([]EventMarketRule, error) {
	var all []EventMarketRule
	err := eachControlDoc(caveats, func(d *Doc) error {
		all = append(all, d.EventMarkets...)
		if len(all) > MaxEventMarketRules {
			return ctlErr("more than %d event_markets entries on one token", MaxEventMarketRules)
		}
		return nil
	})
	return all, err
}

// CollectEventStakes reads every control caveat on a token, in order, and
// returns its event_stakes entries, each with its ID set.
func CollectEventStakes(caveats []string) ([]EventStakeLimit, error) {
	var all []EventStakeLimit
	err := eachControlDoc(caveats, func(d *Doc) error {
		all = append(all, d.EventStakes...)
		if len(all) > MaxEventStakes {
			return ctlErr("more than %d event_stakes entries on one token", MaxEventStakes)
		}
		return nil
	})
	return all, err
}

func eachControlDoc(caveats []string, f func(*Doc) error) error {
	n := 0
	for _, caveat := range caveats {
		value, ok := strings.CutPrefix(caveat, "scope = ")
		if !ok || !argrules.IsControlScope(value) {
			continue
		}
		n++
		if n > MaxControlCaveats {
			return ctlErr("more than %d control caveats", MaxControlCaveats)
		}
		d, err := ParseScopeValue(value)
		if err != nil {
			return err
		}
		if err := f(d); err != nil {
			return err
		}
	}
	return nil
}

// DescribeEventControls is the plain-words form of event_markets and
// event_stakes, one line per entry. It names no category or series, so an
// agent cannot use it to probe a list.
func DescribeEventControls(markets []EventMarketRule, stakes []EventStakeLimit) []string {
	var out []string
	for _, r := range markets {
		out = append(out, fmt.Sprintf("%s: only markets in the categories or series this token allows", strings.Join(r.Tools, " and ")))
	}
	for _, l := range stakes {
		out = append(out, fmt.Sprintf("%s: at most %s in total on any one event, counting each order at the most it can cost (%s), until the totals are reset",
			strings.Join(l.Tools, " and "), FormatAmount(OrderCostField, l.Max), argrules.EventOrderCountedAs))
	}
	return out
}
