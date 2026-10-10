package argrules

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ControlScopePrefix starts a scope word that carries tool controls with
// state (spending limits over a window; see pkg/toolcontrols). It sits inside
// the reserved "argrules:" namespace on purpose: every verifier that predates
// it, and every verifier that has argument rules but not controls, treats the
// word as an argument-rules word it cannot read, so it denies the token
// instead of dropping the limit (see macaroon.ArgumentRulesScopePrefix).
//
// IsRuleScope is true for these words as well, which keeps every place that
// skips "not a tool name" words correct. Code that parses rule documents
// (Collect, DocumentsSHA256) skips them with IsControlScope.
const ControlScopePrefix = macaroon.ArgumentRulesScopePrefix + "ctl:"

// IsControlScope reports whether a scope caveat value is a tool-controls word
// rather than an argument-rules word.
func IsControlScope(value string) bool {
	return strings.HasPrefix(value, ControlScopePrefix)
}

// The helpers below let pkg/toolcontrols read a numeric argument and validate
// names with exactly the rules argument rules use.

// ValidToolName reports whether name is a tool name a rule or control may use.
func ValidToolName(name string) bool { return toolNameOK(name) }

// ValidFieldName reports whether name is a top-level argument field name a
// rule or control may use.
func ValidFieldName(name string) bool { return fieldNameOK(name) }

// CheckDuplicateKeys fails on a repeated object key at any depth, nesting
// deeper than maxDepth, or data after the first value.
func CheckDuplicateKeys(doc []byte, maxDepth int) error {
	return checkJSONDuplicateKeys(doc, maxDepth)
}

// ParsePlainDecimal is ParseDecimal for callers outside the package.
func ParsePlainDecimal(text string) (*Decimal, error) { return ParseDecimal(text) }

// Rat returns an independent copy of the exact value.
func (d *Decimal) Rat() *big.Rat { return new(big.Rat).Set(d.rat) }

// Why a numeric argument could not be read.
const (
	NumericFieldOK         = ""
	NumericFieldUnreadable = "the call could not be read"
	NumericFieldMissing    = "the field is missing"
	NumericFieldNotNumber  = "the field is not a plain decimal number"
)

// CheckCallShape applies to a tools/call the strict reading of the top level
// that ReadNumericField and argument rules share: params must be valid UTF-8,
// an object with no repeated key at any depth, no top-level key that differs
// from "name" or "arguments" only by letter case, and a name equal to
// toolName. It returns NumericFieldOK or NumericFieldUnreadable. A control
// that needs "the name checked is the name the upstream reads" calls it for
// every call on a token that carries the control, not only for covered tools.
func CheckCallShape(params json.RawMessage, toolName string) string {
	if !utf8.Valid(params) {
		return NumericFieldUnreadable
	}
	top, err := objectOf(params)
	if err != nil || hasDuplicateTopLevelKey(params) {
		return NumericFieldUnreadable
	}
	for k := range top {
		if k != "name" && k != "arguments" && (strings.EqualFold(k, "name") || strings.EqualFold(k, "arguments")) {
			return NumericFieldUnreadable
		}
	}
	var name string
	if raw, ok := top["name"]; !ok || json.Unmarshal(raw, &name) != nil || name != toolName {
		return NumericFieldUnreadable
	}
	if err := checkJSONDuplicateKeys(params, maxCallJSONDepth); err != nil {
		return NumericFieldUnreadable
	}
	return NumericFieldOK
}

// ReadNumericField reads the argument `field` of a tools/call as an exact
// decimal, with the same strictness argument rules apply: params must be valid
// UTF-8, an object with no repeated key at any depth, no top-level key that
// differs from "name" or "arguments" only by letter case, a name equal to
// toolName, and no argument key that differs from field only by letter case.
// A JSON number is read by its literal text, a JSON string must hold a plain
// decimal. The returned reason says why it could not be read; it never holds
// an argument value.
func ReadNumericField(params json.RawMessage, toolName, field string) (*Decimal, string) {
	if why := CheckCallShape(params, toolName); why != NumericFieldOK {
		return nil, why
	}
	top, _ := objectOf(params)
	raw, ok := top["arguments"]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, NumericFieldMissing
	}
	args, err := objectOf(bytes.TrimSpace(raw))
	if err != nil {
		return nil, NumericFieldUnreadable
	}
	for k := range args {
		if k != field && strings.EqualFold(k, field) {
			return nil, NumericFieldUnreadable
		}
	}
	value, present := args[field]
	if !present {
		return nil, NumericFieldMissing
	}
	d, ok := argumentDecimal(bytes.TrimSpace(value))
	if !ok {
		return nil, NumericFieldNotNumber
	}
	return d, NumericFieldOK
}
