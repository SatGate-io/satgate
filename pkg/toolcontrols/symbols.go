package toolcontrols

import (
	"strings"
)

// Symbol caps.
//
// A spending limit may name symbols ("BTC: at most $50 a day"). It then counts
// only the calls whose canonical symbol is on its list. The same coin or stock
// can be written several ways in a call (Robinhood's own description of
// place_crypto_order says "BTC", "ETH" or "BTC-USD", and its quotes accept
// hyphenated and unhyphenated forms), so a cap on the text as sent could be
// walked round by changing the spelling. The canonical form below is computed
// from a fixed table, per tool, and the cap compares canonical forms only.
//
// Only the tools in this table can carry a symbol cap. For any other tool the
// gateway does not know how the service reads a symbol, so a cap naming one is
// refused when the document is parsed instead of guessed at.

// SymbolField is the top-level call argument that holds the symbol.
const SymbolField = "symbol"

// MaxSymbolsPerLimit is how many symbols one limit may list.
const MaxSymbolsPerLimit = 16

const (
	toolCryptoOrder = "place_crypto_order"
	toolEquityOrder = "place_equity_order"
)

// SymbolCapTool reports whether a symbol cap may name tool. The comparison
// ignores letter case, like SpendLimit.Covers.
func SymbolCapTool(tool string) bool {
	return symbolKind(tool) != kindNone
}

type kind int

const (
	kindNone kind = iota
	kindCrypto
	kindEquity
)

func symbolKind(tool string) kind {
	switch {
	case strings.EqualFold(tool, toolCryptoOrder):
		return kindCrypto
	case strings.EqualFold(tool, toolEquityOrder):
		return kindEquity
	}
	return kindNone
}

// asciiTrim removes ASCII white space from both ends. Unicode white space is
// deliberately not removed: text with it fails the character checks below, and
// the call is refused rather than read some other way than the upstream reads it.
func asciiTrim(s string) string {
	return strings.Trim(s, " \t\n\r\v\f")
}

// asciiUpper upper-cases ASCII letters only. strings.ToUpper would also map a
// few non-ASCII letters to ASCII ones (dotless i, long s).
func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}

func cryptoSymbolOK(s string) bool {
	if len(s) < 1 || len(s) > 12 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func equitySymbolOK(s string) bool {
	if len(s) < 1 || len(s) > 10 {
		return false
	}
	if c := s[0]; !(c >= 'A' && c <= 'Z') {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.') {
			return false
		}
	}
	return true
}

// CanonicalSymbol is the canonical form of a symbol sent to tool, and whether
// it is acceptable. The table:
//
//	place_crypto_order  trim, upper-case, strip one trailing "-USD"; the result
//	                    must match ^[A-Z0-9]{1,12}$
//	place_equity_order  trim, upper-case; must match ^[A-Z][A-Z0-9.]{0,9}$
//	any other tool      not accepted
func CanonicalSymbol(tool, raw string) (string, bool) {
	s := asciiUpper(asciiTrim(raw))
	switch symbolKind(tool) {
	case kindCrypto:
		s = strings.TrimSuffix(s, "-USD")
		return s, cryptoSymbolOK(s)
	case kindEquity:
		return s, equitySymbolOK(s)
	}
	return "", false
}

// AmbiguousCryptoStem reports whether canonical, a crypto symbol written
// without a hyphen, ends in "USD" and its stem is on the capped list: "BTCUSD"
// might mean the pair BTC-USD or a coin named BTCUSD, and a cap must not guess.
// It returns the stem so the refusal can ask for it.
func AmbiguousCryptoStem(tool, canonical string, capped []string) (string, bool) {
	if symbolKind(tool) != kindCrypto || !strings.HasSuffix(canonical, "USD") || len(canonical) <= 3 {
		return "", false
	}
	stem := strings.TrimSuffix(canonical, "USD")
	for _, c := range capped {
		if c == stem {
			return stem, true
		}
	}
	return "", false
}

// symbolListed reports whether the limit's list names canonical.
func (l SpendLimit) symbolListed(canonical string) bool {
	for _, s := range l.Symbols {
		if s == canonical {
			return true
		}
	}
	return false
}

// SymbolCheck is how a symbol-capped limit sees one call.
type SymbolCheck int

const (
	// SymbolCounts: the call's symbol is on the list; the limit counts it.
	SymbolCounts SymbolCheck = iota
	// SymbolOther: the call is for another symbol; the limit does not touch it.
	SymbolOther
	// SymbolUnreadable: the symbol is missing, not text, or not canonical-able;
	// the call is refused.
	SymbolUnreadable
	// SymbolAmbiguous: "BTCUSD" with BTC capped; the call is refused and asked
	// to say "BTC".
	SymbolAmbiguous
)

// CheckSymbol decides how a symbol-capped limit treats a call to tool whose
// symbol argument reads raw (present says whether it was readable text). It
// returns the canonical symbol when the limit counts the call, and the stem to
// ask for when the call is ambiguous.
func (l SpendLimit) CheckSymbol(tool, raw string, present bool) (SymbolCheck, string) {
	if len(l.Symbols) == 0 {
		return SymbolCounts, ""
	}
	if !present {
		return SymbolUnreadable, ""
	}
	canon, ok := CanonicalSymbol(tool, raw)
	if !ok {
		return SymbolUnreadable, ""
	}
	if l.symbolListed(canon) {
		return SymbolCounts, canon
	}
	if stem, amb := AmbiguousCryptoStem(tool, canon, l.Symbols); amb {
		return SymbolAmbiguous, stem
	}
	return SymbolOther, ""
}

// SymbolKind names the namespace of the limit's symbols: "crypto" or "equity".
// A symbol cap covers tools of one kind only (see validateSymbols), because
// "BTC" the coin and "BTC" the stock are different things and must not share a
// counter. It is "" for a limit with no symbols.
func (l SpendLimit) SymbolKind() string {
	if len(l.Symbols) == 0 || len(l.Tools) == 0 {
		return ""
	}
	if symbolKind(l.Tools[0]) == kindCrypto {
		return "crypto"
	}
	return "equity"
}
