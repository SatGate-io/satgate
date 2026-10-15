package mcpserver

import "context"

// EnforcementModeResolver returns the enforcement mode that applies to one
// tools/call, given the tenant of the token that made it. A host that serves
// many accounts from one Proxy installs one with SetEnforcementModeResolver,
// so that each account's mode decides that account's calls only.
//
// The answer must be "observe" or "control" (the legacy names shadow,
// chargeback, soft, hard and fiat402 are accepted). Anything else, an empty
// answer included, is read as "control": a call whose mode cannot be
// resolved is blocked on exhaustion, never allowed through.
type EnforcementModeResolver func(ctx context.Context, tenantID string) string

// ctxEnforcementMode carries the mode a tools/call was decided under, so that
// code below the Proxy (a BudgetEnforcer that records the spend) reads the
// same answer the Proxy used.
const ctxEnforcementMode contextKey = "satgate_enforcement_mode"

// EnforcementModeFromContext returns the mode the Proxy resolved for this
// call. It reports false when no resolver is installed (the process-wide mode
// applies) or when the context did not come from a tools/call.
func EnforcementModeFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	mode, ok := ctx.Value(ctxEnforcementMode).(string)
	return mode, ok
}

// SetEnforcementModeResolver installs r. Call it before the Proxy serves
// requests. With no resolver (the default) every call is decided under the
// process-wide mode, as SetEnforcementMode and the configuration set it.
func (p *Proxy) SetEnforcementModeResolver(r EnforcementModeResolver) {
	p.modeResolver = r
}

// enforcementModeFor returns the mode a call by tenantID is decided under.
func (p *Proxy) enforcementModeFor(ctx context.Context, tenantID string) string {
	if p.modeResolver == nil {
		return p.config.Enforcement.Mode
	}
	if mode, ok := EnforcementModeFromContext(ctx); ok {
		return mode
	}
	switch mode := normalizeEnforcementMode(p.modeResolver(ctx, tenantID)); mode {
	case "observe", "control":
		return mode
	}
	return "control"
}

// withResolvedEnforcementMode returns the call's mode and a context that
// carries it. Without a resolver the context is returned as it came.
func (p *Proxy) withResolvedEnforcementMode(ctx context.Context, tenantID string) (context.Context, string) {
	mode := p.enforcementModeFor(ctx, tenantID)
	if p.modeResolver == nil {
		return ctx, mode
	}
	return context.WithValue(ctx, ctxEnforcementMode, mode), mode
}
