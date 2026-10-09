package argrules

// Package argrules lets a token limit what a tool may do, not only which
// tool it may call. Argument rules: a token can limit what a tool may do, not only which tool
// it may call.
//
// A rule names one tool and lists allowed shapes. A call passes a rule when at
// least one shape matches. A shape is a set of conditions on top-level fields
// of params.arguments; every condition in the shape must hold. When a token
// carries several rules for one tool (a holder narrowed it later), every rule
// must pass. A tool with no rule behaves as before.
//
// Wire format (fail closed in older verifiers)
//
// The rules ride inside a scope caveat:
//
//	scope = argrules:v1:<nonce>:<base64url(JSON)>
//
// <nonce> is 32 hex characters (128 random bits), fresh for every rule caveat
// minted, so two tokens with the same rules carry different words.
//
// What this gives an older verifier, exactly: it splits a scope on commas and
// compares each word to the tool name. The rule word is not "*", "api:*" or
// "mcp:*", and it does not end in ":*", so it matches a real tool name only
// by exact equality, and the older code compares the whole word. Every scope
// caveat must allow a tool, so the older verifier denies every real tool on a
// rule-carrying token. The one case it would allow is a tool named exactly
// like the whole word. The word holds 128 random bits that only the token's
// holder and issuer see, so an upstream cannot advertise a name that matches
// a token it has never seen. New runtimes also refuse any upstream tool whose
// name starts with the reserved prefix (see macaroon.ArgumentRulesScopePrefix).
//
// A new caveat name would be ignored by today's verifiers ("Unknown caveats
// are ignored"), which would drop the limit silently, so no new caveat name is
// used. macaroon.Service.Verify also refuses a token carrying this caveat, so
// HTTP routes cannot ignore it.
//
// JSON document (version 1):
//
//	{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[
//	   {"method":{"one_of":["GET"],"ascii_case_insensitive":true}},
//	   {"method":{"one_of":["POST"],"ascii_case_insensitive":true},
//	    "url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}}}
//	]}]}
//
// Conditions: one_of (+ ascii_case_insensitive), max, min, url, absent.
// Any other name, a malformed value, or a size over the limits below is an
// error at mint and at verify.

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Size limits. They are enforced when rules are minted, parsed from a
// caveat, and merged from a token.
const (
	MaxDocBytes        = 8 * 1024 // JSON document in one caveat
	MaxRuleCaveats     = 8        // rule caveats on one token
	MaxRulesPerToken   = 16       // rules over all caveats
	MaxShapesPerRule   = 16
	MaxFieldsPerShape  = 8
	MaxOneOfValues     = 128
	MaxOneOfValueBytes = 256
	MaxURLHosts        = 8
	MaxURLPorts        = 4
	MaxURLPathEntries  = 16
	MaxPathEntryBytes  = 256
	MaxFieldNameBytes  = 64
	MaxToolNameBytes   = 128
	version            = 1
	scopePrefix        = macaroon.ArgumentRulesScopePrefix + "v1:"
	maxJSONNesting     = 16
)

// Rule limits one tool.
type Rule struct {
	Tool   string
	Shapes []Shape
}

// Shape is a set of field conditions. Every condition must hold.
// Conditions are kept in field-name order so encoding is deterministic.
type Shape struct {
	Fields []Condition
}

// Condition limits one top-level field of params.arguments. Exactly one
// of the kinds below is set.
type Condition struct {
	Field string

	OneOf                []string
	ASCIICaseInsensitive bool

	Max *Decimal
	Min *Decimal

	URL *URLCondition

	Absent bool
}

// URLCondition limits an absolute https URL.
type URLCondition struct {
	Hosts        []string // lower-case ASCII, no trailing dot
	Ports        []int    // ports allowed when the URL names one; no port means the https default
	PathPrefixes []string // whole-segment prefixes, e.g. /stores/v1
	PathSuffixes []string // whole-segment suffixes, e.g. /query
}

// ErrInvalid wraps every parse and validation failure.
var ErrInvalid = errors.New("invalid argument rules")

func ruleErr(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// IsRuleScope reports whether a scope caveat value carries rules.
// Anything with the reserved prefix is treated as a rule caveat, so an
// unknown version fails to parse instead of being read as a tool word.
func IsRuleScope(value string) bool {
	return macaroon.IsArgumentRulesScope(value)
}

// ParseJSON parses the version-1 document strictly: exact key
// names, no duplicate keys, no unknown names, size limits.
func ParseJSON(doc []byte) ([]Rule, error) {
	if len(doc) == 0 {
		return nil, ruleErr("empty document")
	}
	if len(doc) > MaxDocBytes {
		return nil, ruleErr("document is larger than %d bytes", MaxDocBytes)
	}
	if err := checkJSONDuplicateKeys(doc, maxJSONNesting); err != nil {
		return nil, ruleErr("%v", err)
	}
	top, err := objectOf(doc)
	if err != nil {
		return nil, ruleErr("document must be an object")
	}
	var rulesRaw json.RawMessage
	sawV := false
	for k, v := range top {
		switch k {
		case "v":
			if string(bytes.TrimSpace(v)) != "1" {
				return nil, ruleErr("unsupported version")
			}
			sawV = true
		case "rules":
			rulesRaw = v
		default:
			return nil, ruleErr("unknown document key %q", clip(k))
		}
	}
	if !sawV || rulesRaw == nil {
		return nil, ruleErr("document needs v and rules")
	}
	var list []json.RawMessage
	if err := json.Unmarshal(rulesRaw, &list); err != nil {
		return nil, ruleErr("rules must be a list")
	}
	if len(list) == 0 {
		return nil, ruleErr("rules is empty")
	}
	if len(list) > MaxRulesPerToken {
		return nil, ruleErr("more than %d rules", MaxRulesPerToken)
	}
	out := make([]Rule, 0, len(list))
	for _, raw := range list {
		rule, err := parseRule(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, nil
}

func parseRule(raw json.RawMessage) (Rule, error) {
	obj, err := objectOf(raw)
	if err != nil {
		return Rule{}, ruleErr("a rule must be an object")
	}
	var rule Rule
	var shapesRaw json.RawMessage
	for k, v := range obj {
		switch k {
		case "tool":
			if err := json.Unmarshal(v, &rule.Tool); err != nil {
				return Rule{}, ruleErr("tool must be a string")
			}
		case "shapes":
			shapesRaw = v
		default:
			return Rule{}, ruleErr("unknown rule key %q", clip(k))
		}
	}
	if !toolNameOK(rule.Tool) {
		return Rule{}, ruleErr("tool name is not allowed")
	}
	if shapesRaw == nil {
		return Rule{}, ruleErr("rule for %q needs shapes", rule.Tool)
	}
	var shapes []json.RawMessage
	if err := json.Unmarshal(shapesRaw, &shapes); err != nil {
		return Rule{}, ruleErr("shapes must be a list")
	}
	if len(shapes) == 0 {
		// An empty list would allow nothing. A rule that cannot pass is
		// a mistake, not a feature.
		return Rule{}, ruleErr("rule for %q has no shapes", rule.Tool)
	}
	if len(shapes) > MaxShapesPerRule {
		return Rule{}, ruleErr("rule for %q has more than %d shapes", rule.Tool, MaxShapesPerRule)
	}
	for _, s := range shapes {
		shape, err := parseShape(rule.Tool, s)
		if err != nil {
			return Rule{}, err
		}
		rule.Shapes = append(rule.Shapes, shape)
	}
	return rule, nil
}

func parseShape(tool string, raw json.RawMessage) (Shape, error) {
	obj, err := objectOf(raw)
	if err != nil {
		return Shape{}, ruleErr("a shape must be an object")
	}
	if len(obj) == 0 {
		// An empty shape would match every call and make the rule a no-op.
		return Shape{}, ruleErr("a shape for %q has no conditions", tool)
	}
	if len(obj) > MaxFieldsPerShape {
		return Shape{}, ruleErr("a shape for %q has more than %d fields", tool, MaxFieldsPerShape)
	}
	names := make([]string, 0, len(obj))
	for k := range obj {
		names = append(names, k)
	}
	sort.Strings(names)
	var shape Shape
	for _, name := range names {
		if !fieldNameOK(name) {
			return Shape{}, ruleErr("field name %q is not allowed", clip(name))
		}
		cond, err := parseCondition(tool, name, obj[name])
		if err != nil {
			return Shape{}, err
		}
		shape.Fields = append(shape.Fields, cond)
	}
	return shape, nil
}

func parseCondition(tool, field string, raw json.RawMessage) (Condition, error) {
	obj, err := objectOf(raw)
	if err != nil {
		return Condition{}, ruleErr("conditions for %s.%s must be an object", tool, field)
	}
	cond := Condition{Field: field}
	kinds := 0
	sawFlag := false
	for k, v := range obj {
		switch k {
		case "one_of":
			kinds++
			if err := json.Unmarshal(v, &cond.OneOf); err != nil {
				return Condition{}, ruleErr("one_of for %s.%s must be a list of strings", tool, field)
			}
			if len(cond.OneOf) == 0 || len(cond.OneOf) > MaxOneOfValues {
				return Condition{}, ruleErr("one_of for %s.%s needs 1 to %d values", tool, field, MaxOneOfValues)
			}
			for _, s := range cond.OneOf {
				if len(s) > MaxOneOfValueBytes {
					return Condition{}, ruleErr("a one_of value for %s.%s is too long", tool, field)
				}
			}
		case "ascii_case_insensitive":
			sawFlag = true
			if err := json.Unmarshal(v, &cond.ASCIICaseInsensitive); err != nil {
				return Condition{}, ruleErr("ascii_case_insensitive for %s.%s must be true or false", tool, field)
			}
		case "max":
			kinds++
			d, err := parseConditionDecimal(v)
			if err != nil {
				return Condition{}, ruleErr("max for %s.%s is not a plain decimal", tool, field)
			}
			cond.Max = d
		case "min":
			kinds++
			d, err := parseConditionDecimal(v)
			if err != nil {
				return Condition{}, ruleErr("min for %s.%s is not a plain decimal", tool, field)
			}
			cond.Min = d
		case "url":
			kinds++
			u, err := parseURLCondition(tool, field, v)
			if err != nil {
				return Condition{}, err
			}
			cond.URL = u
		case "absent":
			kinds++
			if err := json.Unmarshal(v, &cond.Absent); err != nil || !cond.Absent {
				return Condition{}, ruleErr("absent for %s.%s must be true", tool, field)
			}
		default:
			return Condition{}, ruleErr("unknown condition %q for %s.%s", clip(k), tool, field)
		}
	}
	if sawFlag && cond.OneOf == nil {
		return Condition{}, ruleErr("ascii_case_insensitive for %s.%s needs one_of", tool, field)
	}
	if kinds == 0 {
		return Condition{}, ruleErr("%s.%s has no condition", tool, field)
	}
	// max and min may share a field (a range). Every other pairing is
	// ambiguous and refused.
	if kinds > 1 {
		onlyRange := cond.OneOf == nil && cond.URL == nil && !cond.Absent && cond.Max != nil && cond.Min != nil
		if !onlyRange {
			return Condition{}, ruleErr("%s.%s mixes conditions; only max with min may share a field", tool, field)
		}
		if cond.Min.Cmp(cond.Max) > 0 {
			return Condition{}, ruleErr("min is above max for %s.%s", tool, field)
		}
	}
	return cond, nil
}

func parseConditionDecimal(raw json.RawMessage) (*Decimal, error) {
	// Rule bounds are written as JSON strings so no tool rounds them.
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return ParseDecimal(s)
}

func parseURLCondition(tool, field string, raw json.RawMessage) (*URLCondition, error) {
	obj, err := objectOf(raw)
	if err != nil {
		return nil, ruleErr("url for %s.%s must be an object", tool, field)
	}
	u := &URLCondition{}
	for k, v := range obj {
		switch k {
		case "hosts":
			if err := json.Unmarshal(v, &u.Hosts); err != nil {
				return nil, ruleErr("hosts for %s.%s must be a list of strings", tool, field)
			}
		case "ports":
			if err := json.Unmarshal(v, &u.Ports); err != nil {
				return nil, ruleErr("ports for %s.%s must be a list of numbers", tool, field)
			}
		case "path_prefixes":
			if err := json.Unmarshal(v, &u.PathPrefixes); err != nil {
				return nil, ruleErr("path_prefixes for %s.%s must be a list of strings", tool, field)
			}
		case "path_suffixes":
			if err := json.Unmarshal(v, &u.PathSuffixes); err != nil {
				return nil, ruleErr("path_suffixes for %s.%s must be a list of strings", tool, field)
			}
		default:
			return nil, ruleErr("unknown url condition %q for %s.%s", clip(k), tool, field)
		}
	}
	if len(u.Hosts) == 0 || len(u.Hosts) > MaxURLHosts {
		return nil, ruleErr("url for %s.%s needs 1 to %d hosts", tool, field, MaxURLHosts)
	}
	for _, h := range u.Hosts {
		if !ruleHostOK(h) {
			return nil, ruleErr("a url host for %s.%s is not a lower-case ASCII domain name", tool, field)
		}
	}
	if len(u.Ports) > MaxURLPorts {
		return nil, ruleErr("url for %s.%s lists more than %d ports", tool, field, MaxURLPorts)
	}
	for _, p := range u.Ports {
		if p < 1 || p > 65535 {
			return nil, ruleErr("a url port for %s.%s is out of range", tool, field)
		}
	}
	if len(u.PathPrefixes) > MaxURLPathEntries || len(u.PathSuffixes) > MaxURLPathEntries {
		return nil, ruleErr("url for %s.%s lists too many paths", tool, field)
	}
	for _, p := range u.PathPrefixes {
		if !rulePathOK(p, true) {
			return nil, ruleErr("a url path prefix for %s.%s is not a clean path", tool, field)
		}
	}
	for _, p := range u.PathSuffixes {
		if !rulePathOK(p, false) {
			return nil, ruleErr("a url path suffix for %s.%s is not a clean path", tool, field)
		}
	}
	return u, nil
}

func ruleHostOK(h string) bool {
	if h == "" || len(h) > 253 || h[len(h)-1] == '.' || h[0] == '.' {
		return false
	}
	labels := strings.Split(h, ".")
	for _, l := range labels {
		if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
		for i := 0; i < len(l); i++ {
			c := l[i]
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	// A name whose last label is all digits is an IP literal.
	last := labels[len(labels)-1]
	allDigits := true
	for i := 0; i < len(last); i++ {
		if last[i] < '0' || last[i] > '9' {
			allDigits = false
		}
	}
	return !allDigits
}

// rulePathOK accepts a clean slash path. The prefix "/" alone (every path)
// is allowed for prefixes only.
func rulePathOK(p string, allowRoot bool) bool {
	if len(p) == 0 || len(p) > MaxPathEntryBytes || p[0] != '/' {
		return false
	}
	if p == "/" {
		return allowRoot
	}
	if _, ok := splitCleanPath(p); !ok {
		return false
	}
	return !strings.Contains(p, "%")
}

func toolNameOK(name string) bool {
	if name == "" || len(name) > MaxToolNameBytes {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '_' || c == '-' || c == '.' || c == ':' || c == '/':
		default:
			return false
		}
	}
	return name != "*" && !strings.HasSuffix(name, ":*")
}

func fieldNameOK(name string) bool {
	if name == "" || len(name) > MaxFieldNameBytes {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}

func objectOf(raw []byte) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("not an object")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &m); err != nil || m == nil {
		return nil, errors.New("not an object")
	}
	return m, nil
}

func clip(s string) string {
	if len(s) > 32 {
		return s[:32] + "..."
	}
	return s
}

// ---- encoding ----

// Marshal is the canonical version-1 JSON document.
func Marshal(rules []Rule) ([]byte, error) {
	if len(rules) == 0 {
		return nil, ruleErr("no rules")
	}
	docRules := make([]any, 0, len(rules))
	for _, r := range rules {
		shapes := make([]any, 0, len(r.Shapes))
		for _, s := range r.Shapes {
			shape := map[string]any{}
			for _, f := range s.Fields {
				cond := map[string]any{}
				switch {
				case f.OneOf != nil:
					cond["one_of"] = f.OneOf
					if f.ASCIICaseInsensitive {
						cond["ascii_case_insensitive"] = true
					}
				}
				if f.Max != nil {
					cond["max"] = f.Max.String()
				}
				if f.Min != nil {
					cond["min"] = f.Min.String()
				}
				if f.URL != nil {
					u := map[string]any{"hosts": f.URL.Hosts}
					if len(f.URL.Ports) > 0 {
						u["ports"] = f.URL.Ports
					}
					if len(f.URL.PathPrefixes) > 0 {
						u["path_prefixes"] = f.URL.PathPrefixes
					}
					if len(f.URL.PathSuffixes) > 0 {
						u["path_suffixes"] = f.URL.PathSuffixes
					}
					cond["url"] = u
				}
				if f.Absent {
					cond["absent"] = true
				}
				shape[f.Field] = cond
			}
			shapes = append(shapes, shape)
		}
		docRules = append(docRules, map[string]any{"tool": r.Tool, "shapes": shapes})
	}
	doc, err := json.Marshal(map[string]any{"v": version, "rules": docRules})
	if err != nil {
		return nil, err
	}
	// Round trip so only rules that parse can be minted.
	if _, err := ParseJSON(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// nonceHexLen is the length of the nonce in a rule word: 16 random bytes in hex.
const nonceHexLen = 32

// newNonce returns 128 fresh random bits as 32 lowercase hex characters.
func newNonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("argument rules: no random nonce available: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// ScopeValue is the scope caveat value that carries rules. Each call draws a
// fresh 128-bit nonce, so the same rules give a different word every time.
func ScopeValue(rules []Rule) (string, error) {
	doc, err := Marshal(rules)
	if err != nil {
		return "", err
	}
	nonce, err := newNonce()
	if err != nil {
		return "", err
	}
	return scopePrefix + nonce + ":" + base64.RawURLEncoding.EncodeToString(doc), nil
}

// Caveat is the full caveat string, "scope = argrules:v1:...".
func Caveat(rules []Rule) (string, error) {
	v, err := ScopeValue(rules)
	if err != nil {
		return "", err
	}
	return "scope = " + v, nil
}

// ParseScopeValue decodes one rule scope value.
func ParseScopeValue(value string) ([]Rule, error) {
	if !strings.HasPrefix(value, scopePrefix) {
		return nil, ruleErr("unsupported rule version")
	}
	rest := strings.TrimPrefix(value, scopePrefix)
	nonce, enc, ok := strings.Cut(rest, ":")
	if !ok || len(nonce) != nonceHexLen {
		return nil, ruleErr("rule word has no nonce")
	}
	for i := 0; i < len(nonce); i++ {
		c := nonce[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, ruleErr("rule word has no nonce")
		}
	}
	if len(enc) > base64.RawURLEncoding.EncodedLen(MaxDocBytes) {
		return nil, ruleErr("document is larger than %d bytes", MaxDocBytes)
	}
	doc, err := base64.RawURLEncoding.Strict().DecodeString(enc)
	if err != nil {
		return nil, ruleErr("rule payload is not base64url")
	}
	return ParseJSON(doc)
}

// Describe is the plain-words form shown to a token owner.
func Describe(rules []Rule) []string {
	var out []string
	for _, r := range rules {
		var alts []string
		for _, s := range r.Shapes {
			var parts []string
			for _, f := range s.Fields {
				parts = append(parts, describeCondition(f))
			}
			alts = append(alts, strings.Join(parts, " and "))
		}
		out = append(out, fmt.Sprintf("%s: allowed only when %s", r.Tool, strings.Join(alts, "; or when ")))
	}
	return out
}

func describeCondition(f Condition) string {
	switch {
	case f.Absent:
		return fmt.Sprintf("%s is not sent", f.Field)
	case f.OneOf != nil:
		suffix := ""
		if f.ASCIICaseInsensitive {
			suffix = " (any letter case)"
		}
		return fmt.Sprintf("%s is %s%s", f.Field, strings.Join(f.OneOf, " or "), suffix)
	case f.URL != nil:
		s := fmt.Sprintf("%s is an https address on %s", f.Field, strings.Join(f.URL.Hosts, " or "))
		if len(f.URL.PathPrefixes) > 0 {
			s += " with a path starting " + strings.Join(f.URL.PathPrefixes, " or ")
		}
		if len(f.URL.PathSuffixes) > 0 {
			s += " with a path ending " + strings.Join(f.URL.PathSuffixes, " or ")
		}
		return s
	case f.Max != nil && f.Min != nil:
		return fmt.Sprintf("%s is between %s and %s", f.Field, f.Min, f.Max)
	case f.Max != nil:
		return fmt.Sprintf("%s is at most %s", f.Field, f.Max)
	case f.Min != nil:
		return fmt.Sprintf("%s is at least %s", f.Field, f.Min)
	}
	return f.Field
}

// Collect reads every rule caveat on a token, in order. allows reports
// whether one scope caveat value allows a tool (the MCP matchScope). A rule
// naming a tool that the scope caveats before it do not allow is refused, so a
// rule can never exist for a tool the token cannot call. Scope caveats added
// after a rule may still narrow the token. Size limits apply to the total.
func Collect(caveats []string, allows func(scope, tool string) bool) ([]Rule, error) {
	var all []Rule
	var scopes []string
	ruleCaveats := 0
	for _, caveat := range caveats {
		if !strings.HasPrefix(caveat, "scope = ") {
			continue
		}
		value := strings.TrimPrefix(caveat, "scope = ")
		if !IsRuleScope(value) {
			scopes = append(scopes, value)
			continue
		}
		ruleCaveats++
		if ruleCaveats > MaxRuleCaveats {
			return nil, ruleErr("more than %d rule caveats", MaxRuleCaveats)
		}
		rules, err := ParseScopeValue(value)
		if err != nil {
			return nil, err
		}
		for _, r := range rules {
			for _, s := range scopes {
				if !allows(s, r.Tool) {
					return nil, ruleErr("a rule names tool %q, which the token scope does not allow", r.Tool)
				}
			}
		}
		all = append(all, rules...)
		if len(all) > MaxRulesPerToken {
			return nil, ruleErr("more than %d rules on one token", MaxRulesPerToken)
		}
	}
	return all, nil
}

// DocumentsSHA256 identifies the rule documents a token carries, so a receipt
// can name the limit without holding the token. It reads every rule caveat in
// token order, parses it, and takes its canonical document (the bytes Marshal
// returns, which is what a minting service acknowledges). It returns "" when
// the caveats hold no rule. See HashDocuments for the digest.
func DocumentsSHA256(caveats []string) (string, error) {
	var docs [][]byte
	for _, caveat := range caveats {
		if !strings.HasPrefix(caveat, "scope = ") {
			continue
		}
		value := strings.TrimPrefix(caveat, "scope = ")
		if !IsRuleScope(value) {
			continue
		}
		rules, err := ParseScopeValue(value)
		if err != nil {
			return "", err
		}
		doc, err := Marshal(rules)
		if err != nil {
			return "", err
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return "", nil
	}
	return HashDocuments(docs), nil
}

// HashDocuments is the lower-case hex SHA-256 of the canonical documents as one
// compact JSON array in the order given: "[" doc1 "," doc2 "]", each document
// byte for byte as Marshal produced it.
func HashDocuments(docs [][]byte) string {
	h := sha256.New()
	h.Write([]byte("["))
	for i, d := range docs {
		if i > 0 {
			h.Write([]byte(","))
		}
		h.Write(d)
	}
	h.Write([]byte("]"))
	return hex.EncodeToString(h.Sum(nil))
}

// Denial says why a call was refused by argument rules. It never holds an
// argument value: Field is a field name taken from the rule.
type Denial struct {
	Field  string
	Reason string // one of the Reason constants
}

func (d *Denial) Error() string {
	if d.Field == "" {
		return d.Reason
	}
	return d.Reason + " (" + d.Field + ")"
}

const (
	ReasonArgsUnreadable = "arguments could not be checked"
	ReasonRulesInvalid   = "token argument rules are not valid"
	ReasonNoShape        = "arguments are outside what this token allows"

	// ReasonScriptToolRule: the token carries a rule for a tool that runs
	// scripts. That rule cannot be enforced, so the call is refused.
	ReasonScriptToolRule = "this tool runs scripts, so a limit on it cannot be enforced"

	// maxCallJSONDepth bounds nesting while looking for duplicate keys.
	maxCallJSONDepth = 64
)
