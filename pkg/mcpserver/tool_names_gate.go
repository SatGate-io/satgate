package mcpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// Rule 7: no new unguarded path.
//
// A token's spending limits, approval rules, argument rules and tool allow-list
// are written for tool names. When a tool is listed under a name other than its
// own (a clash, or an own name agent apps do not accept) that name is new: no
// rule written before it existed can name it. A token that carries a spending
// limit or an approval rule must not reach it by default, or the second
// broker's order tool would be the one tool the limits do not cover.
//
// So a token with any spending limit or approval rule is refused on a renamed
// tool unless one of its rules, or its allow-list, names that exact tool.
// Names compare without regard to case, as SpendLimit.Covers does. A token
// with no money rules is not affected.

// DenialCodeToolNotCovered is the stable code of a call refused because the
// tool is not covered by the token's spending limits.
const DenialCodeToolNotCovered = "TOOL_NOT_COVERED_BY_LIMITS"

// ToolNotCoveredMessage is what the agent is told.
const ToolNotCoveredMessage = "This tool isn't covered by your spending limits. Ask the owner to add it."

// ErrToolNotCovered is returned by RenamedToolRefusal.
var ErrToolNotCovered = errors.New(ToolNotCoveredMessage)

// RenamedToolRefusal returns ErrToolNotCovered when token carries a spending
// limit or approval rule and none of its rules or its allow-list names tool. A
// nil token and a token without money rules give nil. A token whose controls
// cannot be read is refused: a control that cannot be checked is not skipped.
func RenamedToolRefusal(token *TokenInfo, tool string) error {
	if token == nil {
		return nil
	}
	limits, err := token.AllSpendLimits()
	if err != nil {
		return ErrToolNotCovered
	}
	approvals, err := token.AllApprovalRules()
	if err != nil {
		return ErrToolNotCovered
	}
	if len(limits) == 0 && len(approvals) == 0 {
		return nil
	}
	for _, l := range limits {
		if l.Covers(tool) {
			return nil
		}
	}
	for _, a := range approvals {
		if a.Covers(tool) {
			return nil
		}
	}
	rules := token.ArgumentRules
	if token.Raw != nil {
		var cerr error
		if rules, cerr = argrules.Collect(token.Raw.Caveats, matchScope); cerr != nil {
			return ErrToolNotCovered
		}
	}
	for _, r := range rules {
		if strings.EqualFold(r.Tool, tool) {
			return nil
		}
	}
	if scopeNamesTool(token, tool) {
		return nil
	}
	return ErrToolNotCovered
}

// scopeNamesTool reports whether an allow-list on the token lists tool by its
// exact name (a wildcard such as "*" or "db:*" does not name it).
func scopeNamesTool(token *TokenInfo, tool string) bool {
	var scopes []string
	if token.Raw != nil {
		for _, c := range token.Raw.Caveats {
			v, ok := strings.CutPrefix(c, "scope = ")
			if !ok || argrules.IsRuleScope(v) {
				continue
			}
			scopes = append(scopes, v)
		}
	}
	if len(scopes) == 0 && token.Scope != "" {
		scopes = append(scopes, token.Scope)
	}
	for _, scope := range scopes {
		for _, s := range strings.Split(scope, ",") {
			s = strings.TrimSpace(s)
			if len(s) > 37 && s[36] == ':' && s[8] == '-' && s[13] == '-' {
				s = s[37:]
			}
			if s == tool {
				return true
			}
		}
	}
	return false
}

// RenamedToolGate is the CallGate for rule 7. IsRenamed says whether a name
// the agent sent is a renamed tool of the caller's tenant (for the public
// manager: UpstreamManager.IsRenamedTool). Chain it with the other gates.
type RenamedToolGate struct {
	IsRenamed func(ctx context.Context, tool string) bool
}

// NewRenamedToolGate returns the gate over a lookup of renamed tools.
func NewRenamedToolGate(isRenamed func(ctx context.Context, tool string) bool) *RenamedToolGate {
	return &RenamedToolGate{IsRenamed: isRenamed}
}

// Admit implements CallGate.
func (g *RenamedToolGate) Admit(ctx context.Context, call GateCall) (*GateAdmission, error) {
	if g == nil || g.IsRenamed == nil || !g.IsRenamed(ctx, call.Tool) {
		return nil, nil
	}
	if RenamedToolRefusal(call.Token, call.Tool) == nil {
		return nil, nil
	}
	return &GateAdmission{Refusal: &GateRefusal{
		Code:    DenialCodeToolNotCovered,
		Message: ToolNotCoveredMessage,
		Detail:  GateReceiptDetail{DenialCode: DenialCodeToolNotCovered},
	}}, nil
}

// RenamedToolRouter is implemented by an UpstreamRouter that can say whether a
// name is a renamed tool of a tenant. The default router (one shared manager)
// is handled without it.
type RenamedToolRouter interface {
	IsRenamedToolForTenant(ctx context.Context, tenantID, tool string) bool
}

func (p *Proxy) isRenamedTool(ctx context.Context, tenantID, tool string) bool {
	if r, ok := p.router.(RenamedToolRouter); ok {
		return r.IsRenamedToolForTenant(ctx, tenantID, tool)
	}
	if _, ok := p.router.(*defaultRouter); ok && p.upstream != nil {
		return p.upstream.IsRenamedTool(tool)
	}
	return false
}

// renamedToolRefusal refuses a call to a renamed tool (rule 7) with the same
// signed policy denial a gate refusal gets. It returns nil when the call may
// go on. A token with no money rules is never refused here.
func (p *Proxy) renamedToolRefusal(ctx context.Context, req *Request, tokenInfo *TokenInfo, tool string) *Response {
	if !p.isRenamedTool(ctx, tokenInfo.TenantID, tool) {
		return nil
	}
	if RenamedToolRefusal(tokenInfo, tool) == nil {
		return nil
	}
	return p.gateRefusalResponse(ctx, req, tokenInfo, tool, &GateRefusal{
		Code:    DenialCodeToolNotCovered,
		Message: ToolNotCoveredMessage,
		Detail:  GateReceiptDetail{DenialCode: DenialCodeToolNotCovered},
	})
}

// SetHealthDetail sets a function whose non-empty answer is added to the
// gateway's /health reply as "detail" (for example "tool names unavailable").
// It never changes the status: a gateway whose tool names cannot be read is
// still up. Without one, /health is unchanged.
func (p *Proxy) SetHealthDetail(fn func() string) { p.healthDetail = fn }
