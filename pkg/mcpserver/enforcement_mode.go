package mcpserver

import (
	"context"

	"github.com/rs/zerolog/log"
)

// EnforcementModeResolver returns the enforcement mode that applies to one
// tools/call, given the tenant of the token that made it. A host that serves
// many accounts from one Proxy installs one with SetEnforcementModeResolver,
// so that each account's mode decides that account's calls only.
//
// The answer must be "observe" or "control" (the legacy names shadow,
// chargeback, soft, hard and fiat402 are accepted). Anything else, an empty
// answer included, is read as "control": a call whose mode cannot be
// resolved is blocked on exhaustion, never allowed through. A resolver that
// panics is read the same way (see resolveEnforcementMode).
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
	switch mode := p.resolveEnforcementMode(ctx, tenantID); mode {
	case "observe", "control":
		return mode
	}
	return "control"
}

// resolveEnforcementMode calls the host's resolver at the resolution boundary.
// A resolver that panics decides the call as control: the panic is contained
// here, so it never leaves tools/call, and the one log line says that it
// happened without the panic value, which could carry a tenant's data or a
// credential. The call is then refused on exhaustion like any other control
// call.
func (p *Proxy) resolveEnforcementMode(ctx context.Context, tenantID string) (mode string) {
	defer func() {
		if recover() != nil {
			log.Error().Msg("enforcement mode resolver panicked; deciding the call as control")
			mode = "control"
		}
	}()
	return normalizeEnforcementMode(p.modeResolver(ctx, tenantID))
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
