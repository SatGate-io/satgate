package argrules

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// DenialCode is the stable denial code for a call refused by argument rules.
const DenialCode = "TOOL_ARGUMENT_DENIED"

// Check applies rules to a tools/call. params is the exact params object the
// upstream will receive. It returns nil when the call may go on: either no
// rule names toolName, or every rule for toolName passes.
//
// When the token has rules, params must be valid UTF-8, an object with no
// repeated top-level key, and with no key that differs from
// "name" or "arguments" only by letter case (Go's decoder merges those; a strict
// upstream does not), and its name must equal toolName. When a rule names the
// tool, a repeated key at any depth is refused too. Any of these is a denial,
// so the values checked here are the values the upstream reads. Nothing is
// rewritten.
func Check(rules []Rule, toolName string, params json.RawMessage) *Denial {
	if len(rules) == 0 {
		return nil
	}
	// A rule on a tool that runs scripts cannot be enforced (see RunsScripts).
	// Mint never produces one, so a token that carries one was built some other
	// way. It is not trusted at all: every call on it is refused, so the rule
	// can never look like a limit that is being applied.
	if ScriptToolRuleError(rules) != nil {
		return &Denial{Reason: ReasonScriptToolRule}
	}
	// The top level is read strictly whenever the token has any rule: the
	// tool name checked must be the tool name the upstream reads.
	if !utf8.Valid(params) {
		return &Denial{Reason: ReasonArgsUnreadable}
	}
	top, err := objectOf(params)
	if err != nil || hasDuplicateTopLevelKey(params) {
		return &Denial{Reason: ReasonArgsUnreadable}
	}
	for k := range top {
		if k != "name" && k != "arguments" && (strings.EqualFold(k, "name") || strings.EqualFold(k, "arguments")) {
			return &Denial{Reason: ReasonArgsUnreadable}
		}
	}
	var name string
	if raw, ok := top["name"]; !ok || json.Unmarshal(raw, &name) != nil || name != toolName {
		return &Denial{Reason: ReasonArgsUnreadable}
	}
	applies := false
	for _, r := range rules {
		if r.Tool == toolName {
			applies = true
			break
		}
	}
	if !applies {
		return nil
	}
	// A rule applies: refuse a repeated key at any depth, so the arguments
	// checked are the arguments the upstream reads.
	if err := checkJSONDuplicateKeys(params, maxCallJSONDepth); err != nil {
		return &Denial{Reason: ReasonArgsUnreadable}
	}
	var args map[string]json.RawMessage
	if raw, ok := top["arguments"]; ok {
		trimmed := bytes.TrimSpace(raw)
		if !bytes.Equal(trimmed, []byte("null")) {
			if args, err = objectOf(trimmed); err != nil {
				return &Denial{Reason: ReasonArgsUnreadable}
			}
		}
	}
	// A key that differs from a limited field only by letter case would be
	// the same field to an upstream that reads names case-insensitively.
	for _, rule := range rules {
		if rule.Tool != toolName {
			continue
		}
		for k := range args {
			if rule.hasFieldFoldedOnly(k) {
				return &Denial{Reason: ReasonArgsUnreadable}
			}
		}
	}
	// Every rule for the tool must pass.
	for _, rule := range rules {
		if rule.Tool != toolName {
			continue
		}
		if d := rule.check(args); d != nil {
			return d
		}
	}
	return nil
}

// hasDuplicateTopLevelKey reports a repeated key in the outermost object
// (nested values are skipped).
func hasDuplicateTopLevelKey(doc []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(doc))
	if _, err := dec.Token(); err != nil {
		return true
	}
	seen := map[string]struct{}{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return true
		}
		key, ok := kt.(string)
		if !ok {
			return true
		}
		if _, dup := seen[key]; dup {
			return true
		}
		seen[key] = struct{}{}
		if err := skipJSONValue(dec); err != nil {
			return true
		}
	}
	return false
}

func skipJSONValue(dec *json.Decoder) error {
	var v json.RawMessage
	return dec.Decode(&v)
}

// check passes when at least one shape matches. On failure it names the
// first failing field (in the shape's own field-name order) of the shape that
// came closest, meaning the shape with the most conditions held. Every
// condition of every shape is evaluated, so a condition that fails early does
// not hide the ones after it. On a tie the earliest shape in rule order wins.
// The pass/deny verdict depends only on whether some shape has no failing
// condition; the count only picks which field is reported.
func (r Rule) check(args map[string]json.RawMessage) *Denial {
	var best *Denial
	bestHeld := -1
	for _, shape := range r.Shapes {
		held := 0
		var failed string
		var allowed string
		for _, cond := range shape.Fields {
			if cond.holds(args) {
				held++
				continue
			}
			if failed == "" {
				failed = cond.Field
				allowed = cond.AgentText() // from the rule, never the call
			}
		}
		if failed == "" {
			return nil
		}
		if held > bestHeld {
			bestHeld = held
			best = &Denial{Field: failed, Reason: ReasonNoShape, Allowed: allowed}
		}
	}
	if best == nil {
		best = &Denial{Reason: ReasonNoShape}
	}
	return best
}

func (c Condition) holds(args map[string]json.RawMessage) bool {
	raw, present := args[c.Field]
	if c.Absent {
		return !present
	}
	if !present {
		return false
	}
	raw = bytes.TrimSpace(raw)
	switch {
	case c.OneOf != nil:
		s, ok := jsonString(raw)
		if !ok {
			return false
		}
		return oneOfMatches(c.OneOf, s, c.ASCIICaseInsensitive)
	case c.URL != nil:
		s, ok := jsonString(raw)
		return ok && matchArgumentURL(s, c.URL)
	case c.Max != nil || c.Min != nil:
		d, ok := argumentDecimal(raw)
		if !ok {
			return false
		}
		if c.Max != nil && d.Cmp(c.Max) > 0 {
			return false
		}
		if c.Min != nil && d.Cmp(c.Min) < 0 {
			return false
		}
		return true
	}
	return false
}

func jsonString(raw []byte) (string, bool) {
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// argumentDecimal reads a JSON number by its literal text, or a JSON string
// holding a plain decimal. No float64 is involved.
func argumentDecimal(raw []byte) (*Decimal, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var text string
	switch raw[0] {
	case '"':
		s, ok := jsonString(raw)
		if !ok {
			return nil, false
		}
		text = s
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		text = string(raw) // the literal as sent; the grammar check refuses 1e3
	default:
		return nil, false
	}
	d, err := ParseDecimal(text)
	if err != nil {
		return nil, false
	}
	return d, true
}

func oneOfMatches(list []string, v string, foldASCII bool) bool {
	if !foldASCII {
		return containsString(list, v)
	}
	for i := 0; i < len(v); i++ {
		if v[i] >= 0x80 {
			return false
		}
	}
	lv := asciiLower(v)
	for _, s := range list {
		if asciiLower(s) == lv {
			return true
		}
	}
	return false
}

// checkJSONDuplicateKeys walks doc once and fails on a repeated object key at
// any depth (keys compared after JSON unescaping), on nesting deeper than
// maxDepth, and on anything after the first value.
func checkJSONDuplicateKeys(doc []byte, maxDepth int) error {
	dec := json.NewDecoder(bytes.NewReader(doc))
	if err := walkJSONValue(dec, 0, maxDepth); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("extra data after the JSON value")
	}
	return nil
}

func walkJSONValue(dec *json.Decoder, depth, maxDepth int) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	if depth+1 > maxDepth {
		return errors.New("JSON nested too deeply")
	}
	switch delim {
	case '{':
		seen := map[string]struct{}{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := kt.(string)
			if !ok {
				return errors.New("bad object key")
			}
			if _, dup := seen[key]; dup {
				return fmt.Errorf("duplicate key")
			}
			seen[key] = struct{}{}
			if err := walkJSONValue(dec, depth+1, maxDepth); err != nil {
				return err
			}
		}
		_, err := dec.Token() // }
		return err
	case '[':
		for dec.More() {
			if err := walkJSONValue(dec, depth+1, maxDepth); err != nil {
				return err
			}
		}
		_, err := dec.Token() // ]
		return err
	}
	return nil
}

// hasFieldFoldedOnly reports whether key equals a field the rule limits when
// letter case is ignored, but not exactly.
func (r Rule) hasFieldFoldedOnly(key string) bool {
	for _, shape := range r.Shapes {
		for _, c := range shape.Fields {
			if c.Field != key && strings.EqualFold(c.Field, key) {
				return true
			}
		}
	}
	return false
}
