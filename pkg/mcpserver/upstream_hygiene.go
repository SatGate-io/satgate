package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"
)

// What the proxy hands to an upstream it starts or calls: the environment of
// a stdio upstream's process, and the caller's SatGate token.

// stdioBaseEnv is the fixed list of variables of the proxy's environment
// every stdio upstream receives (each only if the proxy has it). It is what a
// program needs to start and run: the search path, the home and user, the
// locale, the temp dir, the time zone, the terminal type and the system's
// TLS trust store, plus the variables Windows needs to run a program. Any
// other variable reaches an upstream only if its configuration names it in
// EnvPassthrough or sets it in Env.
var stdioBaseEnv = [...]string{
	"PATH",
	"HOME",
	"USER",
	"LOGNAME",
	"SHELL",
	"LANG",
	"LANGUAGE",
	"LC_ALL",
	"LC_CTYPE",
	"LC_MESSAGES",
	"TMPDIR",
	"TMP",
	"TEMP",
	"TZ",
	"TERM",
	"SSL_CERT_FILE",
	"SSL_CERT_DIR",
	// Windows
	"SystemRoot",
	"SystemDrive",
	"ComSpec",
	"PATHEXT",
	"windir",
	"USERPROFILE",
	"APPDATA",
	"LOCALAPPDATA",
	"ProgramData",
}

// stdioEnv is the environment of a stdio upstream's process: stdioBaseEnv,
// then cfg.EnvPassthrough, each with the value lookup gives (a name lookup
// does not have is left out), then cfg.Env, which wins. The list is never nil
// (a nil exec.Cmd.Env would mean the proxy's whole environment). The entries
// from the proxy's environment come first, sorted, and the cfg.Env entries
// after them, sorted, so the same configuration always gives the same list,
// and where names are compared without regard to case (Windows, whose
// os/exec keeps the last of two such entries) cfg.Env still wins.
func stdioEnv(cfg UpstreamConfig, lookup func(string) (string, bool)) []string {
	inherited := make(map[string]string, len(stdioBaseEnv)+len(cfg.EnvPassthrough))
	pass := func(name string) {
		if name == "" || strings.ContainsAny(name, "=\x00") {
			return
		}
		if _, set := cfg.Env[name]; set {
			return
		}
		if v, ok := lookup(name); ok {
			inherited[name] = v
		}
	}
	for _, name := range stdioBaseEnv {
		pass(name)
	}
	for _, name := range cfg.EnvPassthrough {
		pass(name)
	}
	env := make([]string, 0, len(inherited)+len(cfg.Env))
	for k, v := range inherited {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	configured := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		configured = append(configured, k+"="+v)
	}
	sort.Strings(configured)
	return append(env, configured...)
}

// verifiedCredentialKey carries, on a request's context, the credential the
// proxy verified for that request. It is set only where Authenticator.Verify
// accepted the credential (authenticate, or verifyConnectionCredential), and
// it is the only value an upstream configured with ForwardCallerToken is
// given: params are never read for it, so nothing a caller writes into
// _meta can take its place.
type verifiedCredentialKey struct{}

func withVerifiedCredential(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, verifiedCredentialKey{}, token)
}

func verifiedCredentialFrom(ctx context.Context) string {
	t, _ := ctx.Value(verifiedCredentialKey{}).(string)
	return t
}

// verifyConnectionCredential is ctx for a method passed to the default
// upstream without going through authenticate. When that upstream is
// configured with ForwardCallerToken and the request carries no verified
// credential yet, the connection's own credential (the Authorization header,
// or ?token= with server.allowQueryToken; never params) is verified, and
// checked for revocation, and only if both pass is it carried on. Otherwise
// ctx is returned as it is, and the upstream gets no credential.
func (p *Proxy) verifyConnectionCredential(ctx context.Context) context.Context {
	if verifiedCredentialFrom(ctx) != "" || p.upstream == nil || !p.upstream.defaultForwardsCallerToken() {
		return ctx
	}
	tok := transportTokenFrom(ctx)
	if tok == "" || p.auth == nil {
		return ctx
	}
	info, err := p.auth.Verify(ctx, tok)
	if err != nil || info == nil {
		return ctx
	}
	if p.revocation != nil && info.BudgetID != "" {
		rctx := ctx
		if info.TenantID != "" {
			rctx = context.WithValue(ctx, CtxTenantID, info.TenantID)
		}
		if p.revocation.IsRevoked(rctx, info.BudgetID) {
			return ctx
		}
	}
	return withVerifiedCredential(ctx, tok)
}

// withoutMetaToken returns params with the token removed from _meta. Member
// names are matched the way encoding/json matches them (case-insensitively),
// since that is how authenticate reads the token, so no spelling of it is
// passed on, and every _meta member is cleaned if the name repeats. A _meta
// left empty is removed. Everything else is kept byte for byte: the other
// members, their order, their names as written and the space around them.
// Params with no token in _meta, or that are not a JSON object, are returned
// unchanged.
func withoutMetaToken(params json.RawMessage) json.RawMessage {
	obj, ok := parseObject(params)
	if !ok {
		return params
	}
	changed := false
	kept := obj.members[:0:0]
	for _, mb := range obj.members {
		if !strings.EqualFold(mb.name, "_meta") {
			kept = append(kept, mb)
			continue
		}
		inner, isObject := parseObject(mb.value)
		if !isObject {
			kept = append(kept, mb) // not an object: no token can be read from it
			continue
		}
		innerKept := inner.members[:0:0]
		for _, im := range inner.members {
			if !strings.EqualFold(im.name, "token") {
				innerKept = append(innerKept, im)
			}
		}
		if len(innerKept) == len(inner.members) {
			kept = append(kept, mb)
			continue
		}
		changed = true
		if len(innerKept) == 0 {
			continue // _meta held only the token
		}
		mb.value = inner.write(innerKept)
		kept = append(kept, mb)
	}
	if !changed {
		return params
	}
	return obj.write(kept)
}

// jsonObject is a JSON object as written: the bytes up to and including '{',
// its members, and the bytes from the end of the last member on.
type jsonObject struct {
	prefix  []byte
	members []objectMember
	suffix  []byte
}

// objectMember is one member of a JSON object: its decoded name, the bytes
// before it (space and the separating comma), the bytes from the start of its
// name to the start of its value (the name as written, the colon and the
// space around it), and its value exactly as written.
type objectMember struct {
	name  string
	sep   []byte
	head  []byte
	value json.RawMessage
}

// parseObject reads the JSON object raw, members in order, repeated names
// included. ok is false when raw is not exactly one JSON object.
func parseObject(raw json.RawMessage) (obj jsonObject, ok bool) {
	if len(raw) == 0 {
		return jsonObject{}, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return jsonObject{}, false
	}
	prev := int(dec.InputOffset())
	obj.prefix = raw[:prev]
	exact := true
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return jsonObject{}, false
		}
		name, isString := tok.(string)
		if !isString {
			return jsonObject{}, false
		}
		nameEnd := int(dec.InputOffset())
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return jsonObject{}, false
		}
		valueEnd := int(dec.InputOffset())
		valueStart := valueEnd - len(value)
		mb := objectMember{name: name, value: value}
		if nameStart := bytes.IndexByte(raw[prev:nameEnd], '"'); exact && nameStart >= 0 &&
			valueStart >= nameEnd && bytes.Equal(raw[valueStart:valueEnd], value) {
			mb.sep, mb.head = raw[prev:prev+nameStart], raw[prev+nameStart:valueStart]
		} else {
			exact = false
		}
		obj.members = append(obj.members, mb)
		prev = valueEnd
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return jsonObject{}, false
	}
	if _, err := dec.Token(); err != io.EOF {
		return jsonObject{}, false
	}
	obj.suffix = raw[prev:]
	if !exact {
		// The decoder's offsets did not match the input (not expected): the
		// object is written from its decoded names and values instead, so a
		// token is still removed, only the spacing is not kept.
		obj.prefix, obj.suffix = []byte("{"), []byte("}")
		for i := range obj.members {
			name, _ := json.Marshal(obj.members[i].name)
			obj.members[i].sep, obj.members[i].head = []byte(","), append(name, ':')
		}
	}
	return obj, true
}

// write writes the object with members in place of its own, each member as
// written. The first member loses the comma before it, if it had one.
func (o jsonObject) write(members []objectMember) json.RawMessage {
	var b bytes.Buffer
	b.Write(o.prefix)
	for i, mb := range members {
		if i == 0 {
			b.Write(bytes.ReplaceAll(mb.sep, []byte(","), nil))
		} else {
			b.Write(mb.sep)
		}
		b.Write(mb.head)
		b.Write(mb.value)
	}
	b.Write(o.suffix)
	return b.Bytes()
}

// defaultForwardsCallerToken reports whether the default upstream is
// configured with ForwardCallerToken.
func (m *UpstreamManager) defaultForwardsCallerToken() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, ok := m.config[m.defaultUp]
	return ok && cfg.ForwardCallerToken
}

// outboundParams is what is sent to the upstream named name: params without
// any _meta.token, and with the request's verified credential (see
// verifiedCredentialKey) put back only when that upstream is configured with
// ForwardCallerToken. A request with no verified credential forwards none.
func (m *UpstreamManager) outboundParams(ctx context.Context, name string, params json.RawMessage) json.RawMessage {
	params = withoutMetaToken(params)
	tok := verifiedCredentialFrom(ctx)
	if tok == "" {
		return params
	}
	m.mu.Lock()
	cfg, ok := m.config[name]
	m.mu.Unlock()
	if !ok || !cfg.ForwardCallerToken {
		return params
	}
	if len(params) == 0 || string(params) == "null" {
		return injectMetaToken(json.RawMessage(`{}`), tok)
	}
	return injectMetaToken(params, tok)
}
